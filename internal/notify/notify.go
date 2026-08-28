package notify

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"mime"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/settings"
)

// 通知事件类型(与前端选项一致)
const (
	EvLogin     = "login"      // 登录成功
	EvLoginFail = "login_fail" // 登录失败
	EvPassword  = "password"   // 修改密码
	EvLicense   = "license"    // 许可证变更
	EvContainer = "container"  // 容器事件
	EvImage     = "image"      // 镜像事件
	EvNetwork   = "network"    // 网络事件
	EvVolume    = "volume"     // 卷事件
	EvSystem    = "system"     // 面板/系统事件
)

// EventActionToType Docker 事件 action → 通知类型(供事件流过滤)
func EventActionToType(action string) string {
	if action == "login" || action == "login_fail" || action == "password" || action == "license" {
		return action
	}
	switch {
	case strings.HasPrefix(action, "container"):
		return EvContainer
	case strings.HasPrefix(action, "image"):
		return EvImage
	case strings.HasPrefix(action, "network"):
		return EvNetwork
	case strings.HasPrefix(action, "volume"):
		return EvVolume
	}
	return EvContainer
}

// Notify 按配置向 Telegram / 邮件发送通知(异步、失败静默)
func Notify(st *settings.Store, eventType, title, body string) {
	s := st.Get()
	var wg syncWait
	if s.TgEnable && contains(s.TgNotifyEvents, eventType) {
		wg.add(func() { sendTelegram(s, title, body) })
	}
	if s.EmailEnable && contains(s.EmailNotifyEvents, eventType) {
		wg.add(func() { sendEmail(s, title, body) })
	}
}

type syncWait struct{}

func (syncWait) add(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("notify error: %v", r)
			}
		}()
		fn()
	}()
}

func contains(list []string, v string) bool {
	for _, x := range list {
		if x == v || x == "*" {
			return true
		}
	}
	return false
}

// ---------------- Telegram ----------------

// tgAPIBase 自定义 Telegram API 服务器(仿 3x-ui telegramAPIServer;空 = 官方)
func tgAPIBase(s *settings.Settings) string {
	if s.TgBotAPIServer != "" {
		return strings.TrimRight(s.TgBotAPIServer, "/")
	}
	return "https://api.telegram.org"
}

func sendTelegram(s *settings.Settings, title, body string) {
	if s.TgBotToken == "" || s.TgAdminChatId == "" {
		return
	}
	text := title
	if body != "" {
		text += "\n" + body
	}
	payload, _ := json.Marshal(map[string]any{
		"chat_id": s.TgAdminChatId,
		"text":    text,
	})
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("%s/bot%s/sendMessage", tgAPIBase(s), s.TgBotToken)
	resp, err := client.Post(url, "application/json", bytes.NewReader(payload))
	if err != nil {
		log.Printf("telegram notify failed: %v", err)
		return
	}
	defer resp.Body.Close()
}

// ---------------- 邮件 (SMTP) ----------------

func sendEmail(s *settings.Settings, title, body string) {
	if s.SmtpHost == "" || s.SmtpFrom == "" {
		return
	}
	port := s.SmtpPort
	if port == 0 {
		port = 25
	}
	addr := fmt.Sprintf("%s:%d", s.SmtpHost, port)
	// 收件人:逗号分隔;为空回退发给自己
	to := splitRecipients(s.SmtpTo)
	if len(to) == 0 {
		to = []string{s.SmtpFrom}
	}
	fromHeader := formatFromHeader(s.SmtpFromName, s.SmtpFrom)
	msg := buildMimeMail(fromHeader, strings.Join(to, ", "), title, body)
	var auth smtp.Auth
	if s.SmtpUser != "" {
		auth = smtp.PlainAuth("", s.SmtpUser, s.SmtpPass, s.SmtpHost)
	}
	if err := sendSMTP(addr, auth, s.SmtpFrom, to, msg, s.SmtpHost, s.SmtpEncryption); err != nil {
		log.Printf("email notify failed: %v", err)
	}
}

// splitRecipients 逗号分隔收件人列表(去空白、去空项)
func splitRecipients(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// formatFromHeader 发件人名称 → From 头显示名(非 ASCII 名称按 RFC 2047 编码)
func formatFromHeader(name, addr string) string {
	if name == "" {
		return addr
	}
	ascii := true
	for _, r := range name {
		if r > 127 {
			ascii = false
			break
		}
	}
	if !ascii {
		name = mime.QEncoding.Encode("utf-8", name)
	}
	return fmt.Sprintf("%s <%s>", name, addr)
}

// sendSMTP 按加密方式发送:ssl = 隐式 TLS(465),starttls = 显式 STARTTLS(587),其余 = 明文(25)
func sendSMTP(addr string, auth smtp.Auth, from string, to []string, msg []byte, host, encryption string) error {
	switch strings.ToLower(encryption) {
	case "ssl":
		return smtpSendTLS(addr, auth, from, to, msg, host, true)
	case "starttls":
		return smtpSendTLS(addr, auth, from, to, msg, host, false)
	default:
		return smtp.SendMail(addr, auth, from, to, msg)
	}
}

func smtpSendTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte, host string, implicit bool) error {
	var conn net.Conn
	var err error
	if implicit {
		conn, err = tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	} else {
		conn, err = net.Dial("tcp", addr)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	if !implicit {
		if err := client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return err
		}
	}
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, r := range to {
		if err := client.Rcpt(r); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(msg); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

// SendTestEmail 立即发送测试邮件(仿 3x-ui testSmtp;同步返回错误)
func SendTestEmail(st *settings.Store) error {
	s := st.Get()
	if s.SmtpHost == "" || s.SmtpFrom == "" {
		return fmt.Errorf("SMTP not configured")
	}
	port := s.SmtpPort
	if port == 0 {
		port = 25
	}
	addr := fmt.Sprintf("%s:%d", s.SmtpHost, port)
	to := splitRecipients(s.SmtpTo)
	if len(to) == 0 {
		to = []string{s.SmtpFrom}
	}
	fromHeader := formatFromHeader(s.SmtpFromName, s.SmtpFrom)
	msg := buildMimeMail(fromHeader, strings.Join(to, ", "), "Docker Manager 测试邮件", "这是一封来自 Docker Manager 的测试邮件,收到说明 SMTP 配置正常。")
	var auth smtp.Auth
	if s.SmtpUser != "" {
		auth = smtp.PlainAuth("", s.SmtpUser, s.SmtpPass, s.SmtpHost)
	}
	return sendSMTP(addr, auth, s.SmtpFrom, to, msg, s.SmtpHost, s.SmtpEncryption)
}

func buildMimeMail(from, to, subject, body string) []byte {
	var b bytes.Buffer
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + to + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(body)
	return b.Bytes()
}
