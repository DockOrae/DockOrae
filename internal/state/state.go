package state

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/DockOrae/DockOrae/internal/agent"
	"github.com/DockOrae/DockOrae/internal/auth"
	"github.com/DockOrae/DockOrae/internal/config"
	"github.com/DockOrae/DockOrae/internal/db"
	"github.com/DockOrae/DockOrae/internal/notify"
	"github.com/DockOrae/DockOrae/internal/settings"
)

// StoredUser users.json 中存储的用户(扩展字段需兼容旧文件缺失的情况)
type StoredUser struct {
	Username           string  `json:"username"`
	PasswordHash       string  `json:"password_hash"`
	Nickname           *string `json:"nickname"`
	Avatar             *string `json:"avatar"`
	MustChangePassword bool    `json:"must_change_password"`
	TotpSecret         *string `json:"totp_secret"`
	PasswordChangedAt  int64   `json:"password_changed_at"` // 安全凭据变更时间戳(SEC-003:旧 JWT 失效依据)
}

// TotpPending 2FA 绑定过程中的临时 secret(启用成功前不落盘)
type TotpPending struct {
	Username string
	Secret   string
}

// EventHub Docker 事件广播(慢消费者直接丢弃,等价旧版 broadcast Lagged 语义)
type EventHub struct {
	mu   sync.Mutex
	subs map[chan agent.EventMessage]struct{}
}

func NewEventHub() *EventHub {
	return &EventHub{subs: make(map[chan agent.EventMessage]struct{})}
}

func (h *EventHub) Subscribe() chan agent.EventMessage {
	ch := make(chan agent.EventMessage, 512)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *EventHub) Unsubscribe(ch chan agent.EventMessage) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

func (h *EventHub) Publish(m agent.EventMessage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs {
		select {
		case ch <- m:
		default: // 慢消费者丢弃
		}
	}
}

type AppState struct {
	Cfg         *config.Config
	Settings    *settings.Store
	DB          *db.DB // SQLite(users/settings/events)
	UsersMu     sync.Mutex
	Users       []StoredUser
	Events      *EventHub
	ComposeDir  string
	AvatarDir   string
	TotpMu      sync.Mutex
	TotpPending *TotpPending
	Agent       *agent.Client // 宿主机控制平面客户端(Agent 未部署时为 nil 可用性检测)
	done        chan struct{}
}

// New 创建 AppState:Agent 未部署时面板仍可启动(相关功能返回不可用)
func New(cfg *config.Config) (*AppState, error) {
	// SQLite 优先(仿 3x-ui:users/settings/events 全量入库);打开失败降级 JSON
	var database *db.DB
	if d, err := db.Open(cfg.DataDir); err == nil {
		database = d
	} else {
		log.Printf("sqlite open failed, falling back to JSON: %v", err)
	}
	st, err := settings.Load(cfg.DataDir, os.Getenv("PORT"), database)
	if err != nil {
		return nil, err
	}
	composeDir := filepath.Join(cfg.DataDir, "compose")
	avatarDir := filepath.Join(cfg.DataDir, "avatars")
	_ = os.MkdirAll(composeDir, 0o755)
	_ = os.MkdirAll(avatarDir, 0o755)

	as := &AppState{
		Cfg:        cfg,
		Settings:   st,
		DB:         database,
		Events:     NewEventHub(),
		ComposeDir: composeDir,
		AvatarDir:  avatarDir,
		done:       make(chan struct{}),
	}
	if err := as.initUsers(); err != nil {
		return nil, err
	}
	as.ResetAdminPasswordIfMarked()
	as.initAgentClient()
	as.SpawnEventWatcher()
	return as, nil
}

// initAgentClient 初始化 Agent 客户端:
//   - token 未配置时自动生成并持久化(settings),同时写入共享目录 agent.token 供 Agent 读取
//   - socket 路径:AGENT_SOCKET 环境变量 > settings.AgentSocket > 默认 /run/dockorae/agent.sock
func (s *AppState) initAgentClient() {
	sock := os.Getenv("AGENT_SOCKET")
	if sock == "" {
		sock = s.Settings.Get().AgentSocket
	}
	if sock == "" {
		sock = agent.DefaultSocket
	}
	token := s.Settings.Get().AgentToken
	if token == "" {
		token = agent.GenerateToken()
		_ = s.Settings.Update(map[string]any{"agentToken": token})
	}
	// 写共享 token 文件(尽力而为;Agent 与面板同宿主机时生效)
	_ = agent.WriteTokenFile(token)
	s.Agent = agent.New(sock, token)
	if agent.SocketExists(sock) {
		log.Printf("agent: connected to %s", sock)
	} else {
		log.Printf("agent: socket %s 不存在(Agent 未部署,相关功能将返回不可用)", sock)
	}
}

// initUsers 从 SQLite 加载用户;库为空时迁移旧 users.json,再没有则创建默认 admin
func (s *AppState) initUsers() error {
	if s.DB != nil {
		usersFile := filepath.Join(s.Cfg.DataDir, "users.json")
		if _, err := s.DB.ImportUsers(usersFile); err != nil {
			log.Printf("users import failed: %v", err)
		}
		list, err := s.DB.ListUsers()
		if err == nil {
			s.Users = make([]StoredUser, 0, len(list))
			for _, u := range list {
				s.Users = append(s.Users, StoredUser{
					Username:           u.Username,
					PasswordHash:       u.PasswordHash,
					Nickname:           u.Nickname,
					Avatar:             u.Avatar,
					MustChangePassword: u.MustChangePassword,
					TotpSecret:         u.TotpSecret,
					PasswordChangedAt:  u.PasswordChangedAt,
				})
			}
			if len(s.Users) > 0 {
				return nil
			}
		} else {
			log.Printf("users load failed: %v", err)
		}
		// 库为空:创建默认 admin 并入库
		hash, err := auth.HashPassword("123456")
		if err != nil {
			return err
		}
		admin := StoredUser{Username: "admin", PasswordHash: hash, MustChangePassword: true}
		s.Users = []StoredUser{admin}
		if err := s.DB.UpsertUser(db.User{
			Username: admin.Username, PasswordHash: admin.PasswordHash, MustChangePassword: true,
		}); err != nil {
			return err
		}
		log.Println("Created default user admin / 123456 (change password after first login)")
		return nil
	}

	// 无 SQLite(降级路径):维持旧 JSON 行为
	usersFile := filepath.Join(s.Cfg.DataDir, "users.json")
	if raw, err := os.ReadFile(usersFile); err == nil {
		var v struct {
			Users []StoredUser `json:"users"`
		}
		if json.Unmarshal(raw, &v) == nil && v.Users != nil {
			s.Users = v.Users
			return nil
		}
	}
	hash, err := auth.HashPassword("123456")
	if err != nil {
		return err
	}
	users := []StoredUser{{
		Username:           "admin",
		PasswordHash:       hash,
		MustChangePassword: true,
	}}
	s.Users = users
	out, _ := json.MarshalIndent(map[string]any{"users": users}, "", "  ")
	if err := os.WriteFile(usersFile, out, 0o600); err != nil {
		return err
	}
	log.Println("Created default user admin / 123456 (change password after first login)")
	return nil
}

func (s *AppState) SaveUsers() error {
	s.UsersMu.Lock()
	users := append([]StoredUser(nil), s.Users...)
	s.UsersMu.Unlock()
	if s.DB != nil {
		list := make([]db.User, 0, len(users))
		for _, u := range users {
			list = append(list, db.User{
				Username:           u.Username,
				PasswordHash:       u.PasswordHash,
				Nickname:           u.Nickname,
				Avatar:             u.Avatar,
				MustChangePassword: u.MustChangePassword,
				TotpSecret:         u.TotpSecret,
				TotpEnabled:        u.TotpSecret != nil,
				PasswordChangedAt:  u.PasswordChangedAt,
			})
		}
		return s.DB.ReplaceUsers(list)
	}
	out, err := json.MarshalIndent(map[string]any{"users": users}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Cfg.DataDir, "users.json"), out, 0o600)
}

// ReloadDB 关闭并重开 SQLite(备份恢复替换 db 文件前必须调用,
// 否则 Windows 下文件被占用无法删除);失败时降级 JSON
func (s *AppState) ReloadDB() {
	if s.Settings != nil {
		_ = s.Settings.Close()
	}
	if s.DB != nil {
		_ = s.DB.Close()
	}
	s.DB = nil
	if d, err := db.Open(s.Cfg.DataDir); err == nil {
		s.DB = d
	} else {
		log.Printf("sqlite reopen failed, falling back to JSON: %v", err)
	}
	if st, err := settings.Load(s.Cfg.DataDir, os.Getenv("PORT"), s.DB); err == nil {
		s.Settings = st
	}
	_ = s.initUsers()
}

func (s *AppState) FindUser(username string) *StoredUser {
	s.UsersMu.Lock()
	defer s.UsersMu.Unlock()
	for i := range s.Users {
		if s.Users[i].Username == username {
			u := s.Users[i]
			return &u
		}
	}
	return nil
}

// ResetAdminPasswordIfMarked 处理安装脚本的密码重置标记(install.sh reset-passwd 配套):
// 数据目录存在 .reset-admin-password 文件时,将 admin 密码重置为 123456 并强制下次修改,
// 然后删除标记。返回是否执行了重置。
// 背景:用户数据已迁移 SQLite,install.sh 旧方案删除 users.json 不再生效,
// 改用"启动标记"由面板自身完成重置(与 initUsers 创建默认 admin 同源逻辑)。
func (s *AppState) ResetAdminPasswordIfMarked() bool {
	marker := filepath.Join(s.Cfg.DataDir, ".reset-admin-password")
	if _, err := os.Stat(marker); err != nil {
		return false
	}
	hash, err := auth.HashPassword("123456")
	if err != nil {
		log.Printf("reset password: hash failed: %v", err)
		return false
	}
	now := time.Now().Unix()
	s.UsersMu.Lock()
	found := false
	for i := range s.Users {
		if s.Users[i].Username == "admin" {
			s.Users[i].PasswordHash = hash
			s.Users[i].MustChangePassword = true
			// 安全凭据变更:已签发 token 全部失效
			s.Users[i].PasswordChangedAt = now
			found = true
			break
		}
	}
	if !found {
		s.Users = append(s.Users, StoredUser{
			Username: "admin", PasswordHash: hash, MustChangePassword: true, PasswordChangedAt: now,
		})
	}
	s.UsersMu.Unlock()
	if err := s.SaveUsers(); err != nil {
		log.Printf("reset password: save failed: %v", err)
		return false
	}
	_ = os.Remove(marker)
	log.Println("admin password has been reset to 123456 (change it after login)")
	return true
}

// SpawnEventWatcher 经 Agent WebSocket 订阅 Docker 事件流,断线 3s 后重连。
// 事件来源为 Agent /v1/docker/events(§5:Docker 全部由 Agent 执行);
// Agent 未部署时静默重试(面板其余功能不受影响)。
func (s *AppState) SpawnEventWatcher() {
	go func() {
		for {
			select {
			case <-s.done:
				return
			default:
			}
			if s.Agent == nil {
				select {
				case <-s.done:
					return
				case <-time.After(3 * time.Second):
				}
				continue
			}
			ctx, cancel := context.WithCancel(context.Background())
			conn, err := s.Agent.DockerEventsWS(ctx)
			if err != nil {
				cancel()
				select {
				case <-s.done:
					return
				case <-time.After(3 * time.Second):
				}
				continue
			}
			for {
				_, data, err := conn.ReadMessage()
				if err != nil {
					break
				}
				var m agent.EventMessage
				if json.Unmarshal(data, &m) != nil {
					continue
				}
				s.Events.Publish(m)
				// 通知:Docker 事件 → TG/邮件(按配置过滤;Notify 内部有界队列,风暴自动丢弃)
				body := fmt.Sprintf("Action: %s\nTarget: %s", m.Action, m.Actor.ID)
				if name := m.Actor.Attributes["name"]; name != "" {
					body += "\nName: " + name
				}
				notify.Notify(s.Settings, notify.EventActionToType(m.Action), "Docker Event: "+m.Action, body)
			}
			_ = conn.Close()
			cancel()
			log.Println("docker events stream disconnected, reconnecting in 3s")
			select {
			case <-s.done:
				return
			case <-time.After(3 * time.Second):
			}
		}
	}()
}

