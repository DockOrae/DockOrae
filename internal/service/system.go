package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/moby/moby/api/types/system"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/auth"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/docker"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/notify"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// PublicUser 用户公开信息(不含密码/totp secret)
func PublicUser(u *state.StoredUser) map[string]any {
	return map[string]any{
		"username":             u.Username,
		"nickname":             u.Nickname,
		"avatar":               u.Avatar,
		"must_change_password": u.MustChangePassword,
		"totp_enabled":         u.TotpSecret != nil,
	}
}

// Login 密码登录:成功返回登录响应;已启用 2FA 返回 totp_required
func Login(st *state.AppState, username, password, clientIP string) (map[string]any, error) {
	u := st.FindUser(username)
	if u == nil || !auth.VerifyPassword(password, u.PasswordHash) {
		if st.DB != nil {
			_ = st.DB.AddEvent("login_fail", username, "invalid credentials", clientIP)
		}
		notify.Notify(st.Settings, notify.EvLoginFail, "面板登录失败",
			"用户: "+username+"\nIP: "+clientIP+"\n时间: "+time.Now().Format("2006-01-02 15:04:05"))
		return nil, NewApiError(401, "login.errPwd")
	}
	// 已启用 2FA:第一步只校验密码,返回 totp_required
	if u.TotpSecret != nil {
		return map[string]any{"totp_required": true, "username": u.Username}, nil
	}
	token := auth.MakeToken(st.Cfg.JWTSecret, u.Username, st.Settings.SessionTTLSeconds())
	if st.DB != nil {
		_ = st.DB.AddEvent("login", u.Username, "password", clientIP)
	}
	notify.Notify(st.Settings, notify.EvLogin, "面板登录成功",
		"用户: "+u.Username+"\nIP: "+clientIP+"\n时间: "+time.Now().Format("2006-01-02 15:04:05"))
	return map[string]any{
		"token": token, "username": u.Username,
		"nickname": u.Nickname, "avatar": u.Avatar,
		"must_change_password": u.MustChangePassword, "totp_enabled": false,
	}, nil
}

// LoginTotp 2FA 第二步:校验 TOTP 码并发 token
func LoginTotp(st *state.AppState, username, code, clientIP string) (map[string]any, error) {
	u := st.FindUser(username)
	if u == nil || u.TotpSecret == nil {
		return nil, NewApiError(401, "login.errTotpCode")
	}
	if !auth.VerifyTotp(*u.TotpSecret, code) {
		if st.DB != nil {
			_ = st.DB.AddEvent("login_fail", username, "invalid totp", clientIP)
		}
		return nil, NewApiError(401, "login.errTotpCode")
	}
	token := auth.MakeToken(st.Cfg.JWTSecret, u.Username, st.Settings.SessionTTLSeconds())
	if st.DB != nil {
		_ = st.DB.AddEvent("login", u.Username, "totp", clientIP)
	}
	return map[string]any{
		"token": token, "username": u.Username,
		"nickname": u.Nickname, "avatar": u.Avatar,
		"must_change_password": u.MustChangePassword, "totp_enabled": true,
	}, nil
}

// Me 当前用户信息
func Me(st *state.AppState, username string) (map[string]any, error) {
	u := st.FindUser(username)
	if u == nil {
		return nil, NewApiError(404, "user.notFound")
	}
	return PublicUser(u), nil
}

// IsValidUsername 用户名合法性(字母数字 _ -)
func IsValidUsername(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

// UpdateProfile 更新昵称/用户名;用户名变更时返回新 token
func UpdateProfile(st *state.AppState, username string, nickname, newUsername *string) (map[string]any, error) {
	var nicknameOut *string
	if nickname != nil {
		t := strings.TrimSpace(*nickname)
		if t != "" {
			if len([]rune(t)) > 32 {
				return nil, BadRequest("user.nicknameTooLong")
			}
			nicknameOut = &t
		}
	}
	var newUsernameOut *string
	if newUsername != nil {
		t := strings.TrimSpace(*newUsername)
		if t != "" {
			if len([]rune(t)) > 32 || !IsValidUsername(t) {
				return nil, BadRequest("user.usernameInvalid")
			}
			st.UsersMu.Lock()
			for i := range st.Users {
				if st.Users[i].Username == t && st.Users[i].Username != username {
					st.UsersMu.Unlock()
					return nil, NewApiError(409, "user.usernameTaken")
				}
			}
			st.UsersMu.Unlock()
			newUsernameOut = &t
		}
	}

	st.UsersMu.Lock()
	var u *state.StoredUser
	for i := range st.Users {
		if st.Users[i].Username == username {
			u = &st.Users[i]
			break
		}
	}
	if u == nil {
		st.UsersMu.Unlock()
		return nil, NewApiError(404, "user.notFound")
	}
	u.Nickname = nicknameOut
	var newToken string
	if newUsernameOut != nil {
		u.Username = *newUsernameOut
		newToken = auth.MakeToken(st.Cfg.JWTSecret, *newUsernameOut, st.Settings.SessionTTLSeconds())
	}
	snapshot := *u
	st.UsersMu.Unlock()
	if err := st.SaveUsers(); err != nil {
		return nil, err
	}
	resp := PublicUser(&snapshot)
	if newToken != "" {
		resp["token"] = newToken
	}
	return resp, nil
}

// ---------------- 壁纸 / 头像 ----------------

const (
	MaxWallpaperBytes = 10 * 1024 * 1024
	MaxAvatarBytes    = 2 * 1024 * 1024
)

func sniffImageExt(b []byte) string {
	switch {
	case len(b) > 8 && bytes.Equal(b[:8], []byte("\x89PNG\r\n\x1a\n")):
		return "png"
	case len(b) > 3 && b[0] == 0xff && b[1] == 0xd8 && b[2] == 0xff:
		return "jpg"
	case len(b) > 6 && (bytes.Equal(b[:6], []byte("GIF87a")) || bytes.Equal(b[:6], []byte("GIF89a"))):
		return "gif"
	case len(b) > 12 && bytes.Equal(b[:4], []byte("RIFF")) && bytes.Equal(b[8:12], []byte("WEBP")):
		return "webp"
	}
	return ""
}

func MimeByExt(name string) string {
	if t := mime.TypeByExtension(filepath.Ext(name)); t != "" {
		return t
	}
	return "application/octet-stream"
}

// WallpaperSave 上传登录页壁纸(base64 → data/wallpaper.jpg)
func WallpaperSave(st *state.AppState, dataB64 string) error {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(dataB64))
	if err != nil || len(data) == 0 {
		return BadRequest("wallpaper.invalid")
	}
	if len(data) > MaxWallpaperBytes {
		return BadRequest("wallpaper.tooLarge")
	}
	if sniffImageExt(data) == "" {
		return BadRequest("wallpaper.type")
	}
	path := filepath.Join(st.Cfg.DataDir, "wallpaper.jpg")
	return os.WriteFile(path, data, 0o644)
}

// WallpaperPath 壁纸文件路径(未设置返回 error)
func WallpaperPath(st *state.AppState) (string, error) {
	path := filepath.Join(st.Cfg.DataDir, "wallpaper.jpg")
	if _, err := os.Stat(path); err != nil {
		return "", NewApiError(404, "wallpaper.notFound")
	}
	return path, nil
}

// UploadAvatar 上传头像,返回新文件名
func UploadAvatar(st *state.AppState, username, dataB64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(strings.TrimSpace(dataB64))
	if err != nil || len(data) == 0 {
		return "", BadRequest("user.avatarInvalid")
	}
	if len(data) > MaxAvatarBytes {
		return "", BadRequest("user.avatarTooLarge")
	}
	ext := sniffImageExt(data)
	if ext == "" {
		return "", BadRequest("user.avatarType")
	}
	id := make([]byte, 16)
	_, _ = rand.Read(id)
	fileName := hex.EncodeToString(id) + "." + ext
	path := filepath.Join(st.AvatarDir, fileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}

	st.UsersMu.Lock()
	var u *state.StoredUser
	for i := range st.Users {
		if st.Users[i].Username == username {
			u = &st.Users[i]
			break
		}
	}
	if u == nil {
		st.UsersMu.Unlock()
		_ = os.Remove(path)
		return "", NewApiError(404, "user.notFound")
	}
	old := u.Avatar
	u.Avatar = &fileName
	st.UsersMu.Unlock()
	if err := st.SaveUsers(); err != nil {
		return "", err
	}
	if old != nil && *old != "" {
		_ = os.Remove(filepath.Join(st.AvatarDir, *old))
	}
	return fileName, nil
}

// SafeFileName 文件名合法性(字母数字 . - _)
func SafeFileName(s string) bool {
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

// AvatarData 读取头像文件
func AvatarData(st *state.AppState, file string) ([]byte, error) {
	if file == "" || len(file) > 64 || !SafeFileName(file) {
		return nil, BadRequest("user.avatarNameInvalid")
	}
	data, err := os.ReadFile(filepath.Join(st.AvatarDir, file))
	if err != nil {
		return nil, NewApiError(404, "user.avatarNotFound")
	}
	return data, nil
}

// ---------------- 修改密码 / TOTP ----------------

// ChangePassword 修改密码
func ChangePassword(st *state.AppState, username, oldPwd, newPwd string) error {
	st.UsersMu.Lock()
	var u *state.StoredUser
	for i := range st.Users {
		if st.Users[i].Username == username {
			u = &st.Users[i]
			break
		}
	}
	if u == nil {
		st.UsersMu.Unlock()
		return NewApiError(404, "user.notFound")
	}
	if !auth.VerifyPassword(oldPwd, u.PasswordHash) {
		st.UsersMu.Unlock()
		return BadRequest("user.oldPwdWrong")
	}
	if len(newPwd) < 6 {
		st.UsersMu.Unlock()
		return BadRequest("user.pwdTooShort")
	}
	if auth.VerifyPassword(newPwd, u.PasswordHash) {
		st.UsersMu.Unlock()
		return BadRequest("user.pwdSame")
	}
	hash, err := auth.HashPassword(newPwd)
	if err != nil {
		st.UsersMu.Unlock()
		return err
	}
	u.PasswordHash = hash
	u.MustChangePassword = false
	st.UsersMu.Unlock()
	return st.SaveUsers()
}

// TotpSetup 生成 2FA secret(临时保存,启用成功前不落盘)
func TotpSetup(st *state.AppState, username, password string) (secret, uri string, err error) {
	u := st.FindUser(username)
	if u == nil {
		return "", "", NewApiError(404, "user.notFound")
	}
	if !auth.VerifyPassword(password, u.PasswordHash) {
		return "", "", BadRequest("user.pwdWrong")
	}
	if u.TotpSecret != nil {
		return "", "", BadRequest("totp.alreadyEnabled")
	}
	secret, err = auth.GenerateTotpSecret()
	if err != nil {
		return "", "", err
	}
	uri = auth.TotpURI(secret, "Docker Manager", u.Username)
	st.TotpMu.Lock()
	st.TotpPending = &state.TotpPending{Username: username, Secret: secret}
	st.TotpMu.Unlock()
	return secret, uri, nil
}

// TotpEnable 校验临时 secret 的 TOTP 码并启用
func TotpEnable(st *state.AppState, username, code string) error {
	st.TotpMu.Lock()
	pending := st.TotpPending
	st.TotpMu.Unlock()
	if pending == nil || pending.Username != username {
		return BadRequest("totp.getKeyFirst")
	}
	if !auth.VerifyTotp(pending.Secret, code) {
		return BadRequest("totp.codeWrong")
	}
	st.UsersMu.Lock()
	var u *state.StoredUser
	for i := range st.Users {
		if st.Users[i].Username == username {
			u = &st.Users[i]
			break
		}
	}
	if u == nil {
		st.UsersMu.Unlock()
		return NewApiError(404, "user.notFound")
	}
	u.TotpSecret = &pending.Secret
	st.UsersMu.Unlock()
	if err := st.SaveUsers(); err != nil {
		return err
	}
	st.TotpMu.Lock()
	st.TotpPending = nil
	st.TotpMu.Unlock()
	return nil
}

// TotpDisable 校验密码 + TOTP 码后关闭 2FA
func TotpDisable(st *state.AppState, username, password, code string) error {
	st.UsersMu.Lock()
	var u *state.StoredUser
	for i := range st.Users {
		if st.Users[i].Username == username {
			u = &st.Users[i]
			break
		}
	}
	if u == nil {
		st.UsersMu.Unlock()
		return NewApiError(404, "user.notFound")
	}
	if !auth.VerifyPassword(password, u.PasswordHash) {
		st.UsersMu.Unlock()
		return BadRequest("user.pwdWrong")
	}
	if u.TotpSecret == nil {
		st.UsersMu.Unlock()
		return BadRequest("totp.notEnabled")
	}
	if !auth.VerifyTotp(*u.TotpSecret, code) {
		st.UsersMu.Unlock()
		return BadRequest("totp.codeWrong")
	}
	u.TotpSecret = nil
	st.UsersMu.Unlock()
	return st.SaveUsers()
}

// ---------------- 杂项 ----------------

// DefaultAccountShow admin 用户存在且仍为默认密码
func DefaultAccountShow(st *state.AppState) bool {
	st.UsersMu.Lock()
	defer st.UsersMu.Unlock()
	for i := range st.Users {
		u := &st.Users[i]
		if u.Username == "admin" && u.MustChangePassword {
			return true
		}
	}
	return false
}

// PublicConfig 前端启动配置(安全入口 basePath)
func PublicConfig(st *state.AppState) map[string]any {
	s := st.Settings.Get()
	return map[string]any{"basePath": s.WebBasePath}
}

// DockerInfo Docker 守护进程信息
func DockerInfo(st *state.AppState, ctx context.Context) (system.Info, error) {
	return docker.DockerInfo(st.Docker, ctx)
}
