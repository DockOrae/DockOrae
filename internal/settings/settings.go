package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Settings 面板设置(存 data_dir/settings.json,仿 3x-ui webSetting)
type Settings struct {
	// 常规
	WebListen       string   `json:"webListen"`       // 面板监听 IP,空 = 监听所有
	WebDomain       string   `json:"webDomain"`       // 面板监听域名
	WebPort         uint16   `json:"webPort"`         // 面板监听端口(重启面板生效)
	WebBasePath     string   `json:"webBasePath"`     // URI 路径,必须以 / 开头结尾
	SessionMaxAge   int      `json:"sessionMaxAge"`   // 会话时长(分钟)
	IPLimitAllowlist []string `json:"ipLimitAllowlist"` // IP 白名单(CIDR 逗号分隔)
	// 证书
	WebCertFile string `json:"webCertFile"` // 面板证书公钥文件路径
	WebKeyFile  string `json:"webKeyFile"`  // 面板证书密钥文件路径
	// 日期和时间
	TimeZone       string `json:"timeZone"`       // 时区
	DatePickerType string `json:"datePickerType"` // 日期选择器日历类型
	// Telegram 机器人
	TgEnable       bool     `json:"tgEnable"`
	TgBotToken     string   `json:"tgBotToken"`
	TgAdminChatId  string   `json:"tgAdminChatId"`
	TgNotifyEvents []string `json:"tgNotifyEvents"`
	// 邮件
	EmailEnable       bool     `json:"emailEnable"`
	SmtpHost          string   `json:"smtpHost"`
	SmtpPort          int      `json:"smtpPort"`
	SmtpUser          string   `json:"smtpUser"`
	SmtpPass          string   `json:"smtpPass"`
	SmtpFrom          string   `json:"smtpFrom"`
	EmailNotifyEvents []string `json:"emailNotifyEvents"`
}

type Store struct {
	mu   sync.Mutex
	path string
	s    *Settings
}

func Default(envPort string) *Settings {
	port := uint16(8080)
	if n, err := strconv.Atoi(envPort); err == nil && n > 0 && n < 65536 {
		port = uint16(n)
	}
	return &Settings{
		WebListen:     "",
		WebDomain:     "",
		WebPort:       port,
		WebBasePath:   "/",
		SessionMaxAge: 7 * 24 * 60, // 7 天(分钟)
		TimeZone:      "Asia/Shanghai",
	}
}

func Load(dataDir string, envPort string) (*Store, error) {
	path := filepath.Join(dataDir, "settings.json")
	s := Default(envPort)
	if raw, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(raw, s) != nil {
			// 解析失败用默认值
		} else {
			normalize(s)
		}
	}
	st := &Store{path: path, s: s}
	_ = st.Save()
	return st, nil
}

func normalize(s *Settings) {
	if s.WebBasePath == "" || !strings.HasPrefix(s.WebBasePath, "/") {
		s.WebBasePath = "/"
	}
	if s.WebBasePath != "/" && !strings.HasSuffix(s.WebBasePath, "/") {
		s.WebBasePath += "/"
	}
	if s.SessionMaxAge <= 0 {
		s.SessionMaxAge = 7 * 24 * 60
	}
	if s.WebPort == 0 {
		s.WebPort = 8080
	}
	if s.TimeZone == "" {
		s.TimeZone = "Asia/Shanghai"
	}
}

func (st *Store) Get() *Settings {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.s
}

func (st *Store) Update(patch map[string]any) error {
	st.mu.Lock()
	defer st.mu.Unlock()
	raw, _ := json.Marshal(patch)
	if err := json.Unmarshal(raw, st.s); err != nil {
		return err
	}
	normalize(st.s)
	return st.Save()
}

func (st *Store) Save() error {
	out, err := json.MarshalIndent(st.s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(st.path, out, 0o600)
}

// SessionTTL 会话时长转 JWT TTL 秒
func (st *Store) SessionTTLSeconds() int64 {
	s := st.Get()
	return int64(s.SessionMaxAge) * 60
}
