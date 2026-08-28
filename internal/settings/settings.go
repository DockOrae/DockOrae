package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/db"
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
	NtpServer      string `json:"ntpServer"`      // NTP 时间同步服务器
	// Telegram 机器人
	TgEnable       bool     `json:"tgEnable"`
	TgBotToken     string   `json:"tgBotToken"`
	TgAdminChatId  string   `json:"tgAdminChatId"`
	TgNotifyEvents []string `json:"tgNotifyEvents"`
	TgRunTime      string   `json:"tgRunTime"`   // 周期报告频率(@every 1h / @daily / crontab 表达式;空 = 不启用)
	TgBotBackup    bool     `json:"tgBotBackup"` // 周期报告附带数据库备份文件
	TgLang         string   `json:"tgLang"`         // Telegram 机器人语言(仿 3x-ui telegramBotLanguage,空=默认)
	TgBotAPIServer string   `json:"tgBotAPIServer"` // 自定义 Telegram API 服务器(仿 3x-ui telegramAPIServer,空=官方)
	// 邮件
	EmailEnable       bool     `json:"emailEnable"`
	SmtpHost          string   `json:"smtpHost"`
	SmtpPort          int      `json:"smtpPort"`
	SmtpUser          string   `json:"smtpUser"`
	SmtpPass          string   `json:"smtpPass"`
	SmtpFrom          string   `json:"smtpFrom"`       // 发件人地址
	SmtpFromName      string   `json:"smtpFromName"`   // 发件人名称(From 头显示名,可选)
	SmtpTo            string   `json:"smtpTo"`         // 收件人,逗号分隔(空 = 发给自己)
	SmtpEncryption    string   `json:"smtpEncryption"` // SMTP 加密:none / ssl / starttls
	EmailNotifyEvents []string `json:"emailNotifyEvents"`
}

type Store struct {
	mu   sync.Mutex
	path string
	s    *Settings
	// SQLite 后端(非空时优先写库;旧 settings.json 首次启动迁移进库)
	conn *db.DB
}

const settingsKey = "main"

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
		NtpServer:     "pool.ntp.org",
	}
}

// Load 加载设置:优先 SQLite(仿 3x-ui),库为空时迁移旧 settings.json。
// shared 为外部已打开的数据库连接(与 users/events 共享),nil 时自行打开。
func Load(dataDir string, envPort string, shared *db.DB) (*Store, error) {
	path := filepath.Join(dataDir, "settings.json")
	st := &Store{path: path, s: Default(envPort)}
	if shared != nil {
		st.conn = shared
	} else if d, err := db.Open(dataDir); err == nil {
		st.conn = d
	}
	if st.conn != nil {
		raw, _ := st.conn.GetSetting(settingsKey)
		if raw != "" {
			// 已入库:直接读库
			if json.Unmarshal([]byte(raw), st.s) == nil {
				normalize(st.s)
				return st, nil
			}
		} else if oldRaw, err := os.ReadFile(path); err == nil && json.Unmarshal(oldRaw, st.s) == nil {
			// 旧 JSON 迁移进库
			normalize(st.s)
			_ = st.saveDB()
			return st, nil
		}
	}
	// 无库且无旧文件:默认值
	if raw, err := os.ReadFile(path); err == nil {
		if json.Unmarshal(raw, st.s) != nil {
			// 解析失败用默认值
		} else {
			normalize(st.s)
		}
	}
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
	if s.NtpServer == "" {
		s.NtpServer = "pool.ntp.org"
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
	// SQLite 优先;JSON 文件保留为兼容快照(备份/迁移兜底)
	if st.conn != nil {
		if err := st.saveDB(); err == nil {
			return nil
		}
	}
	return os.WriteFile(st.path, out, 0o600)
}

// saveDB 写设置到 SQLite(整个 Settings 对象一行 JSON)
func (st *Store) saveDB() error {
	raw, err := json.Marshal(st.s)
	if err != nil {
		return err
	}
	return st.conn.PutSetting(settingsKey, string(raw))
}

// Close 关闭数据库连接(进程退出时)
func (st *Store) Close() error {
	if st.conn != nil {
		return st.conn.Close()
	}
	return nil
}

// SessionTTL 会话时长转 JWT TTL 秒
func (st *Store) SessionTTLSeconds() int64 {
	s := st.Get()
	return int64(s.SessionMaxAge) * 60
}
