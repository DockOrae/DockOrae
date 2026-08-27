package api

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/logger"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/notify"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// LogRing 面板日志环形缓冲(供日志弹窗)
var LogRing = logger.NewRing(2000)

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

// ---------------- 面板设置 (仿 3x-ui GeneralTab / SecurityTab) ----------------

// panelSettings 读取全部设置(脱敏:token 不返回明文)
func panelSettings(c *gin.Context, st *state.AppState) error {
	s := st.Settings.Get()
	out := *s
	if out.TgBotToken != "" {
		out.TgBotToken = maskSecret(out.TgBotToken)
	}
	if out.SmtpPass != "" {
		out.SmtpPass = maskSecret(out.SmtpPass)
	}
	c.JSON(200, out)
	return nil
}

func maskSecret(v string) string {
	if len(v) <= 4 {
		return "****"
	}
	return v[:2] + strings.Repeat("*", len(v)-4) + v[len(v)-2:]
}

// panelSettingsSave 保存设置(补丁合并;3x-ui 改完"重启面板生效")
func panelSettingsSave(c *gin.Context, st *state.AppState) error {
	var patch map[string]any
	if err := c.ShouldBindJSON(&patch); err != nil {
		return BadRequest("err.requestFailed")
	}
	// 空字符串的 token/pass 视为不修改(避免覆盖)
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
	notify.StartReporter(st.Settings, st.Cfg.DataDir)
	c.JSON(200, gin.H{"ok": true, "needRestart": true})
	return nil
}

// ---------------- 面板日志 (环形缓冲) ----------------

// panelLogs 最近日志(仿 3x-ui LogModal;?lines= 控制行数)
func panelLogs(c *gin.Context, st *state.AppState) error {
	lines := 500
	if v := c.Query("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5000 {
			lines = n
		}
	}
	c.JSON(200, gin.H{"logs": LogRing.Lines(lines)})
	return nil
}

// panelEvents 操作/登录事件记录(存 SQLite,仿 3x-ui access log)
func panelEvents(c *gin.Context, st *state.AppState) error {
	if st.DB == nil {
		c.JSON(200, gin.H{"events": []any{}})
		return nil
	}
	list, err := st.DB.RecentEvents(200)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"events": list})
	return nil
}

// panelTestEmail 发送测试邮件(仿 3x-ui testSmtp)
func panelTestEmail(c *gin.Context, st *state.AppState) error {
	if err := notify.SendTestEmail(st.Settings); err != nil {
		return BadRequest("邮件发送失败: " + err.Error())
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// ---------------- 面板配置查看 (仿 3x-ui ConfigModal) ----------------

// panelConfig 面板配置 JSON(settings.json 内容)
func panelConfig(c *gin.Context, st *state.AppState) error {
	raw, err := os.ReadFile(filepath.Join(st.Cfg.DataDir, "settings.json"))
	if err != nil {
		// 没有则返回当前设置对象
		raw, _ = json.MarshalIndent(st.Settings.Get(), "", "  ")
	}
	c.Header("Content-Type", "application/json")
	c.Data(200, "application/json", raw)
	return nil
}

// ---------------- 备份 / 恢复 (仿 3x-ui BackupModal) ----------------

// panelBackup 打包 data 目录为 tar.gz 下载
func panelBackup(c *gin.Context, st *state.AppState) error {
	tmp, err := os.CreateTemp("", "dm-backup-*.tar.gz")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
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
		defer f.Close()
		_, err = io.Copy(tw, f)
		return err
	})
	if walkErr != nil {
		return walkErr
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := gz.Close(); err != nil {
		return err
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return err
	}
	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", `attachment; filename="docker-manager-backup.tar.gz"`)
	_, _ = io.Copy(c.Writer, tmp)
	return nil
}

// panelRestore 上传 tar.gz 恢复数据目录
func panelRestore(c *gin.Context, st *state.AppState) error {
	file, err := c.FormFile("file")
	if err != nil {
		return BadRequest("backup.fileRequired")
	}
	f, err := file.Open()
	if err != nil {
		return BadRequest("backup.fileRequired")
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return BadRequest("backup.invalidFile")
	}
	defer gz.Close()
	tr := tar.NewReader(gz)

	// 先解压到临时目录,校验路径安全后整体替换
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

	// 恢复前:关闭数据库连接(替换 db 文件前必须释放句柄,Windows 下占用无法删除)
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
		// 恢复失败:回滚
		_ = os.RemoveAll(st.Cfg.DataDir)
		_ = os.Rename(rollback, st.Cfg.DataDir)
		return BadRequest("backup.restoreFailed")
	}
	_ = os.RemoveAll(rollback)

	// 恢复后重载数据库与内存状态
	st.ReloadDB()

	notify.Notify(st.Settings, notify.EvSystem, "面板数据恢复", "数据目录已从备份恢复")
	c.JSON(200, gin.H{"ok": true, "needRestart": true})
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
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
		if err != nil {
			return err
		}
		_, err = io.Copy(out, in)
		out.Close()
		return err
	})
}

// ---------------- 重启面板 (仿 3x-ui 重启面板生效) ----------------

func panelRestart(c *gin.Context, st *state.AppState) error {
	c.JSON(200, gin.H{"ok": true})
	RestartPanel()
	return nil
}
