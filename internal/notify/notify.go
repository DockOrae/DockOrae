package notify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"time"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/settings"
)

// 通知事件类型(与前端选项一致)
const (
	EvLogin        = "login"        // 登录成功
	EvLoginFail    = "login_fail"   // 登录失败
	EvPassword     = "password"     // 修改密码
	EvLicense      = "license"      // 许可证变更
	EvContainer    = "container"    // 容器事件
	EvImage        = "image"        // 镜像事件
	EvNetwork      = "network"      // 网络事件
	EvVolume       = "volume"       // 卷事件
	EvSystem       = "system"       // 面板/系统事件
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
		wg.add(func() { sendTelegram(s.TgBotToken, s.TgAdminChatId, title, body) })
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

func sendTelegram(token, chatID, title, body string) {
	if token == "" || chatID == "" {
		return
	}
	text := title
	if body != "" {
		text += "\n" + body
	}
	payload, _ := json.Marshal(map[string]any{
		"chat_id": chatID,
		"text":    text,
	})
	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
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
	msg := buildMimeMail(s.SmtpFrom, title, body)
	var auth smtp.Auth
	if s.SmtpUser != "" {
		auth = smtp.PlainAuth("", s.SmtpUser, s.SmtpPass, s.SmtpHost)
	}
	if err := smtp.SendMail(addr, auth, s.SmtpFrom, []string{s.SmtpFrom}, msg); err != nil {
		log.Printf("email notify failed: %v", err)
	}
}

func buildMimeMail(from, subject, body string) []byte {
	var b bytes.Buffer
	b.WriteString("From: " + from + "\r\n")
	b.WriteString("To: " + from + "\r\n")
	b.WriteString("Subject: " + subject + "\r\n")
	b.WriteString("MIME-Version: 1.0\r\n")
	b.WriteString("Content-Type: text/plain; charset=UTF-8\r\n\r\n")
	b.WriteString(body)
	return b.Bytes()
}
