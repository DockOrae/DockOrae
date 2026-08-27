package state

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/client"
	"github.com/moby/moby/api/types/events"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/auth"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/config"
)

// StoredUser users.json 中存储的用户(扩展字段需兼容旧文件缺失的情况)
type StoredUser struct {
	Username           string  `json:"username"`
	PasswordHash       string  `json:"password_hash"`
	Nickname           *string `json:"nickname"`
	Avatar             *string `json:"avatar"`
	MustChangePassword bool    `json:"must_change_password"`
	TotpSecret         *string `json:"totp_secret"`
}

// TotpPending 2FA 绑定过程中的临时 secret(启用成功前不落盘)
type TotpPending struct {
	Username string
	Secret   string
}

// MonitorCache 监控采样缓存(CPU 使用率需要两次 /proc/stat 采样做差值)
type MonitorCache struct {
	mu  sync.Mutex
	cpu *[2]uint64 // (idle, total)
}

// SampleCPU 用当前 (idle,total) 采样计算 CPU 使用率并缓存(供 api 包调用)
func (s *AppState) SampleCPU(idle, total uint64) float64 {
	s.Monitor.mu.Lock()
	defer s.Monitor.mu.Unlock()
	pct := 0.0
	if s.Monitor.cpu != nil {
		dTotal := total - s.Monitor.cpu[1]
		dIdle := idle - s.Monitor.cpu[0]
		if dTotal > 0 {
			pct = (1.0 - float64(dIdle)/float64(dTotal)) * 100.0
		}
	}
	s.Monitor.cpu = &[2]uint64{idle, total}
	return pct
}

// EventHub Docker 事件广播(慢消费者直接丢弃,等价旧版 broadcast Lagged 语义)
type EventHub struct {
	mu   sync.Mutex
	subs map[chan events.Message]struct{}
}

func NewEventHub() *EventHub {
	return &EventHub{subs: make(map[chan events.Message]struct{})}
}

func (h *EventHub) Subscribe() chan events.Message {
	ch := make(chan events.Message, 512)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *EventHub) Unsubscribe(ch chan events.Message) {
	h.mu.Lock()
	delete(h.subs, ch)
	h.mu.Unlock()
}

func (h *EventHub) Publish(m events.Message) {
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
	Docker       *client.Client
	Cfg          *config.Config
	UsersMu      sync.Mutex
	Users        []StoredUser
	Events       *EventHub
	ComposeDir   string
	AvatarDir    string
	TotpMu       sync.Mutex
	TotpPending  *TotpPending
	Monitor      MonitorCache
	done         chan struct{}
}

// New 创建 AppState:惰性连接(仅请求时真正访问 Docker),无 Docker 也能启动面板
func New(cfg *config.Config) (*AppState, error) {
	docker, err := newDockerClient()
	if err != nil {
		return nil, err
	}
	composeDir := filepath.Join(cfg.DataDir, "compose")
	avatarDir := filepath.Join(cfg.DataDir, "avatars")
	_ = os.MkdirAll(composeDir, 0o755)
	_ = os.MkdirAll(avatarDir, 0o755)

	st := &AppState{
		Docker:     docker,
		Cfg:        cfg,
		Events:     NewEventHub(),
		ComposeDir: composeDir,
		AvatarDir:  avatarDir,
		done:       make(chan struct{}),
	}
	if err := st.initUsers(); err != nil {
		return nil, err
	}
	st.SpawnEventWatcher()
	return st, nil
}

func newDockerClient() (*client.Client, error) {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return client.NewClientWithOpts(client.WithHost(host))
	}
	if runtime.GOOS == "windows" {
		// Windows 本地开发默认 TCP,无 Docker 也能启动面板
		return client.NewClientWithOpts(client.WithHost("tcp://127.0.0.1:2375"))
	}
	return client.NewClientWithOpts()
}

func (s *AppState) initUsers() error {
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
	out, err := json.MarshalIndent(map[string]any{"users": users}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(s.Cfg.DataDir, "users.json"), out, 0o600)
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

// SpawnEventWatcher 轮询 Docker 事件流,断线 3s 后重连(无 Docker 时静默重试)
func (s *AppState) SpawnEventWatcher() {
	go func() {
		for {
			select {
			case <-s.done:
				return
			default:
			}
			res := s.Docker.Events(context.Background(), client.EventsListOptions{})
			for {
				select {
				case m, ok := <-res.Messages:
					if !ok {
						goto reconnect
					}
					s.Events.Publish(m)
				case err, ok := <-res.Err:
					_ = err
					if !ok || err != nil {
						goto reconnect
					}
				}
			}
		reconnect:
			log.Println("docker events stream disconnected, reconnecting in 3s")
			select {
			case <-s.done:
				return
			case <-time.After(3 * time.Second):
			}
		}
	}()
}

// EventToValue 转成旧版前端约定的事件 JSON 结构
func EventToValue(m events.Message) map[string]any {
	return map[string]any{
		"time":             m.Time,
		"action":           string(m.Action),
		"type":             strings.ToLower(string(m.Type)),
		"id":               m.Actor.ID,
		"actor_attributes": m.Actor.Attributes,
	}
}
