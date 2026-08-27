package notify

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/settings"
)

// 周期报告 cron(仿 3x-ui tgRunTime):@every 1h / @daily / @monthly / 自定义 crontab(6 字段,秒启用)
var reporter *cron.Cron

// StartReporter 按设置启动/重启 Telegram 周期报告任务(设置保存后需重新调用)
func StartReporter(st *settings.Store, dataDir string) {
	stopReporter()
	s := st.Get()
	if !s.TgEnable || s.TgBotToken == "" || s.TgAdminChatId == "" || s.TgRunTime == "" {
		return
	}
	c := cron.New(cron.WithSeconds())
	if _, err := c.AddFunc(s.TgRunTime, func() {
		sendPeriodicReport(st, dataDir)
	}); err != nil {
		log.Printf("cron schedule %q invalid: %v", s.TgRunTime, err)
		return
	}
	c.Start()
	reporter = c
	log.Printf("telegram periodic report scheduled: %q", s.TgRunTime)
}

func stopReporter() {
	if reporter != nil {
		reporter.Stop()
		reporter = nil
	}
}

// sendPeriodicReport 周期报告:面板状态摘要;开启数据库备份时附带 data 目录 tar.gz
func sendPeriodicReport(st *settings.Store, dataDir string) {
	s := st.Get()
	title := "📊 Docker Manager 周期报告"
	body := fmt.Sprintf("时间: %s\n版本: v1.0.0\n数据目录: %s",
		time.Now().Format("2006-01-02 15:04:05"), dataDir)
	if s.TgBotBackup {
		path, err := backupData(dataDir)
		if err != nil {
			log.Printf("periodic backup failed: %v", err)
			sendTelegram(s, title, body+"\n(数据库备份生成失败)")
			return
		}
		defer os.Remove(path)
		sendTelegramDocument(s, title+"\n"+body, path)
		return
	}
	sendTelegram(s, title, body)
}

// backupData 打包数据目录(数据库 + 配置)为 tar.gz,返回临时文件路径
func backupData(dataDir string) (string, error) {
	tmp, err := os.CreateTemp("", "dm-db-backup-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	gz := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gz)
	walkErr := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dataDir, path)
		if err != nil {
			return err
		}
		// 排除备份自身与临时文件
		if strings.Contains(rel, "backups") || strings.HasPrefix(rel, ".restore-backup") || strings.Contains(rel, "compose") {
			return nil
		}
		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		hdr.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if walkErr != nil {
		return "", walkErr
	}
	if err := tw.Close(); err != nil {
		return "", err
	}
	if err := gz.Close(); err != nil {
		return "", err
	}
	return tmp.Name(), nil
}

// sendTelegramDocument 通过 Telegram Bot API 发送文档(数据库备份文件)
func sendTelegramDocument(s *settings.Settings, caption, filePath string) {
	if s.TgBotToken == "" || s.TgAdminChatId == "" {
		return
	}
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	_ = w.WriteField("chat_id", s.TgAdminChatId)
	_ = w.WriteField("caption", caption)
	fw, err := w.CreateFormFile("document", filepath.Base(filePath))
	if err != nil {
		log.Printf("telegram document failed: %v", err)
		return
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("telegram document failed: %v", err)
		return
	}
	if _, err := fw.Write(data); err != nil {
		log.Printf("telegram document failed: %v", err)
		return
	}
	_ = w.Close()
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Post(tgAPIBase(s)+"/bot"+s.TgBotToken+"/sendDocument", w.FormDataContentType(), &b)
	if err != nil {
		log.Printf("telegram document failed: %v", err)
		return
	}
	defer resp.Body.Close()
}
