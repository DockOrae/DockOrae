package service

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/logger"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/notify"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// RestartPanel 重启面板进程(新进程替换自身;docker restart:unless-stopped 自动拉起)
func RestartPanel() {
	go func() {
		time.Sleep(800 * time.Millisecond)
		exe, err := os.Executable()
		if err != nil {
			return
		}
		exePath, _ := filepath.Abs(exe)
		cmd := exec.Command(exePath, os.Args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		_ = cmd.Start()
		os.Exit(0)
	}()
}

// ---------------- 面板设置 ----------------

func MaskSecret(v string) string {
	if len(v) <= 4 {
		return "****"
	}
	return v[:2] + strings.Repeat("*", len(v)-4) + v[len(v)-2:]
}

// SettingsGet 读取全部设置(脱敏:token/pass 不返回明文)
func SettingsGet(st *state.AppState) map[string]any {
	s := st.Settings.Get()
	out := *s
	if out.TgBotToken != "" {
		out.TgBotToken = MaskSecret(out.TgBotToken)
	}
	if out.SmtpPass != "" {
		out.SmtpPass = MaskSecret(out.SmtpPass)
	}
	return settingsToMap(&out)
}

func settingsToMap(s any) map[string]any {
	raw, _ := json.Marshal(s)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	return m
}

// SettingsUpdate 保存设置(补丁合并;空串 token/pass 视为不修改)
func SettingsUpdate(st *state.AppState, patch map[string]any) error {
	cur := st.Settings.Get()
	if v, ok := patch["tgBotToken"].(string); ok && v == "" {
		patch["tgBotToken"] = cur.TgBotToken
	}
	if v, ok := patch["smtpPass"].(string); ok && v == "" {
		patch["smtpPass"] = cur.SmtpPass
	}
	if err := st.Settings.Update(patch); err != nil {
		return BadRequest("保存失败: " + err.Error())
	}
	// 设置变更后重启 Telegram 周期报告调度
	notify.StartReporter(st.Settings, st.Cfg.DataDir, DisplayVersion())
	return nil
}

// PanelLogs 最近日志(环形缓冲)
func PanelLogs(lines int) []string {
	return logger.LogRing.Lines(lines)
}

// PanelEvents 操作/登录事件记录
func PanelEvents(st *state.AppState, limit int) ([]any, error) {
	if st.DB == nil {
		return []any{}, nil
	}
	list, err := st.DB.RecentEvents(limit)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(list))
	for i := range list {
		out[i] = list[i]
	}
	return out, nil
}

// TestEmail 发送测试邮件
func TestEmail(st *state.AppState) error {
	return notify.SendTestEmail(st.Settings)
}

// ConfigRaw 面板配置 JSON(设置查看用;token/pass 脱敏,与 SettingsGet 一致——SEC-005)
func ConfigRaw(st *state.AppState) []byte {
	raw, err := os.ReadFile(filepath.Join(st.Cfg.DataDir, "settings.json"))
	if err != nil {
		raw, _ = json.MarshalIndent(st.Settings.Get(), "", "  ")
	}
	// settings.json 中存的是明文 token/pass,查看接口不返回明文
	var v map[string]any
	if json.Unmarshal(raw, &v) == nil {
		if s, ok := v["tgBotToken"].(string); ok && s != "" {
			v["tgBotToken"] = MaskSecret(s)
		}
		if s, ok := v["smtpPass"].(string); ok && s != "" {
			v["smtpPass"] = MaskSecret(s)
		}
		if out, err := json.MarshalIndent(v, "", "  "); err == nil {
			return out
		}
	}
	return raw
}

// BackupToTemp 打包 data 目录为 tar.gz,返回临时文件路径
func BackupToTemp(st *state.AppState) (string, error) {
	tmp, err := os.CreateTemp("", "dm-backup-*.tar.gz")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	gz := gzip.NewWriter(tmp)
	tw := tar.NewWriter(gz)

	walkErr := filepath.Walk(st.Cfg.DataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(st.Cfg.DataDir, path)
		if err != nil {
			return err
		}
		// 跳过备份自身产生的临时文件与运行中的锁文件
		if strings.Contains(rel, "backups") {
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
		// GO-002:循环内显式 Close,避免 defer 堆积文件句柄
		_, err = io.Copy(tw, f)
		f.Close()
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

// RestoreFromReader 从 tar.gz 恢复数据目录(先解压到临时目录校验,整体替换,失败回滚)
func RestoreFromReader(st *state.AppState, src io.Reader) error {
	gz, err := gzip.NewReader(src)
	if err != nil {
		return BadRequest("backup.invalidFile")
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	tmpDir, err := os.MkdirTemp("", "dm-restore-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return BadRequest("backup.invalidFile")
		}
		name := filepath.Clean(hdr.Name)
		if strings.HasPrefix(name, "..") || filepath.IsAbs(name) {
			continue
		}
		target := filepath.Join(tmpDir, name)
		if hdr.Typeflag == tar.TypeDir {
			_ = os.MkdirAll(target, 0o755)
			continue
		}
		_ = os.MkdirAll(filepath.Dir(target), 0o755)
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
		if err != nil {
			continue
		}
		_, _ = io.Copy(out, tr)
		out.Close()
	}

	// 恢复前:关闭数据库连接(替换 db 文件前必须释放句柄)
	st.ReloadDB()
	// 备份当前数据目录(恢复到失败可回滚)
	rollback := filepath.Join(st.Cfg.DataDir, ".restore-backup")
	_ = os.RemoveAll(rollback)
	_ = copyDir(st.Cfg.DataDir, rollback)
	// 清空数据目录并解压
	entries, _ := os.ReadDir(st.Cfg.DataDir)
	for _, e := range entries {
		if e.Name() == ".restore-backup" {
			continue
		}
		_ = os.RemoveAll(filepath.Join(st.Cfg.DataDir, e.Name()))
	}
	if err := copyDir(tmpDir, st.Cfg.DataDir); err != nil {
		_ = os.RemoveAll(st.Cfg.DataDir)
		_ = os.Rename(rollback, st.Cfg.DataDir)
		return BadRequest("backup.restoreFailed")
	}
	_ = os.RemoveAll(rollback)

	// 恢复后重载数据库与内存状态
	st.ReloadDB()
	notify.Notify(st.Settings, notify.EvSystem, "面板数据恢复", "数据目录已从备份恢复")
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		_ = os.MkdirAll(filepath.Dir(target), 0o755)
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			in.Close()
			return err
		}
		// GO-002:循环内显式 Close,避免 defer 堆积文件句柄
		_, err = io.Copy(out, in)
		out.Close()
		in.Close()
		return err
	})
}
