// Package db 提供 SQLite 存储(仿 3x-ui:users / settings / events 全量入库)。
// 使用 modernc.org/sqlite(纯 Go,无 CGO,支持多架构交叉编译)。
package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB SQLite 句柄 + 元信息
type DB struct {
	*sql.DB
	Path string
}

// Open 打开(不存在则创建)数据库并建表;首次启动自动迁移旧 JSON 数据。
func Open(dataDir string) (*DB, error) {
	path := filepath.Join(dataDir, "docker-manager.db")
	d, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	// SQLite 单写者;modernc 驱动建议限制连接数避免锁竞争
	d.SetMaxOpenConns(1)
	d.SetMaxIdleConns(1)

	schema := []string{
		`CREATE TABLE IF NOT EXISTS users (
			username            TEXT PRIMARY KEY,
			password_hash       TEXT NOT NULL,
			nickname            TEXT,
			avatar              TEXT,
			must_change_password INTEGER NOT NULL DEFAULT 0,
			totp_secret         TEXT,
			totp_enabled        INTEGER NOT NULL DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS settings (
			key   TEXT PRIMARY KEY,
			value TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			id     INTEGER PRIMARY KEY AUTOINCREMENT,
			time   INTEGER NOT NULL,
			action TEXT NOT NULL,
			actor  TEXT,
			detail TEXT,
			ip     TEXT
		)`,
	}
	for _, q := range schema {
		if _, err := d.Exec(q); err != nil {
			d.Close()
			return nil, err
		}
	}
	return &DB{DB: d, Path: path}, nil
}

// ---------------- users ----------------

// User users 表行(与旧 users.json StoredUser 同构)
type User struct {
	Username           string
	PasswordHash       string
	Nickname           *string
	Avatar             *string
	MustChangePassword bool
	TotpSecret         *string
	TotpEnabled        bool
}

// ImportUsers 从旧 users.json 导入(仅当 users 表为空)
func (d *DB) ImportUsers(usersFile string) (int, error) {
	var cnt int
	if err := d.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&cnt); err != nil {
		return 0, err
	}
	if cnt > 0 {
		return 0, nil
	}
	raw, err := os.ReadFile(usersFile)
	if err != nil {
		return 0, nil // 没有旧文件不是错误
	}
	var v struct {
		Users []struct {
			Username           string  `json:"username"`
			PasswordHash       string  `json:"password_hash"`
			Nickname           *string `json:"nickname"`
			Avatar             *string `json:"avatar"`
			MustChangePassword bool    `json:"must_change_password"`
			TotpSecret         *string `json:"totp_secret"`
		} `json:"users"`
	}
	if json.Unmarshal(raw, &v) != nil || len(v.Users) == 0 {
		return 0, nil
	}
	tx, err := d.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, u := range v.Users {
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO users (username, password_hash, nickname, avatar, must_change_password, totp_secret, totp_enabled) VALUES (?,?,?,?,?,?,0)`,
			u.Username, u.PasswordHash, u.Nickname, u.Avatar, u.MustChangePassword, u.TotpSecret,
		); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	log.Printf("Migrated %d users from %s into SQLite", len(v.Users), usersFile)
	return len(v.Users), nil
}

// ListUsers 全部用户
func (d *DB) ListUsers() ([]User, error) {
	rows, err := d.Query(`SELECT username, password_hash, nickname, avatar, must_change_password, totp_secret, totp_enabled FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []User
	for rows.Next() {
		var u User
		var mcp, te int
		if err := rows.Scan(&u.Username, &u.PasswordHash, &u.Nickname, &u.Avatar, &mcp, &u.TotpSecret, &te); err != nil {
			return nil, err
		}
		u.MustChangePassword = mcp != 0
		u.TotpEnabled = te != 0
		out = append(out, u)
	}
	return out, rows.Err()
}

// FindUser 按用户名查用户
func (d *DB) FindUser(username string) (*User, error) {
	row := d.QueryRow(`SELECT username, password_hash, nickname, avatar, must_change_password, totp_secret, totp_enabled FROM users WHERE username = ?`, username)
	var u User
	var mcp, te int
	if err := row.Scan(&u.Username, &u.PasswordHash, &u.Nickname, &u.Avatar, &mcp, &u.TotpSecret, &te); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	u.MustChangePassword = mcp != 0
	u.TotpEnabled = te != 0
	return &u, nil
}

// UpsertUser 插入或更新用户
func (d *DB) UpsertUser(u User) error {
	_, err := d.Exec(
		`INSERT INTO users (username, password_hash, nickname, avatar, must_change_password, totp_secret, totp_enabled)
		 VALUES (?,?,?,?,?,?,?)
		 ON CONFLICT(username) DO UPDATE SET
		   password_hash = excluded.password_hash,
		   nickname = excluded.nickname,
		   avatar = excluded.avatar,
		   must_change_password = excluded.must_change_password,
		   totp_secret = excluded.totp_secret,
		   totp_enabled = excluded.totp_enabled`,
		u.Username, u.PasswordHash, u.Nickname, u.Avatar, u.MustChangePassword, u.TotpSecret, u.TotpEnabled,
	)
	return err
}

// ReplaceUsers 全量替换(备份恢复后调用)
func (d *DB) ReplaceUsers(users []User) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM users`); err != nil {
		return err
	}
	for _, u := range users {
		if _, err := tx.Exec(
			`INSERT INTO users (username, password_hash, nickname, avatar, must_change_password, totp_secret, totp_enabled) VALUES (?,?,?,?,?,?,?)`,
			u.Username, u.PasswordHash, u.Nickname, u.Avatar, u.MustChangePassword, u.TotpSecret, u.TotpEnabled,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ---------------- settings ----------------

// GetSetting 读设置 JSON(整个 Settings 对象存一行)
func (d *DB) GetSetting(key string) (string, error) {
	var v string
	err := d.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

// PutSetting 写设置 JSON
func (d *DB) PutSetting(key, value string) error {
	_, err := d.Exec(`INSERT INTO settings (key, value) VALUES (?,?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}

// ---------------- events ----------------

// Event 操作/登录事件记录
type Event struct {
	ID     int64  `json:"id"`
	Time   int64  `json:"time"`
	Action string `json:"action"`
	Actor  string `json:"actor"`
	Detail string `json:"detail"`
	IP     string `json:"ip"`
}

// AddEvent 追加事件(3x-ui access log 语义)
func (d *DB) AddEvent(action, actor, detail, ip string) error {
	_, err := d.Exec(`INSERT INTO events (time, action, actor, detail, ip) VALUES (?,?,?,?,?)`,
		time.Now().Unix(), action, actor, detail, ip)
	return err
}

// RecentEvents 最近 N 条事件
func (d *DB) RecentEvents(limit int) ([]Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	rows, err := d.Query(`SELECT id, time, action, actor, detail, ip FROM events ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		if err := rows.Scan(&e.ID, &e.Time, &e.Action, &e.Actor, &e.Detail, &e.IP); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
