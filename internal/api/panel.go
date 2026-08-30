package api

import (
	"io"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/DockOrae/DockOrae/internal/service"
)

func panelSettings(c *gin.Context, d *Deps) error {
	c.JSON(200, service.SettingsGet(d.St))
	return nil
}

func panelSettingsSave(c *gin.Context, d *Deps) error {
	var patch map[string]any
	if err := c.ShouldBindJSON(&patch); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.SettingsUpdate(d.St, patch); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true, "needRestart": true})
	return nil
}

func panelLogs(c *gin.Context, d *Deps) error {
	lines := 500
	if v := c.Query("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5000 {
			lines = n
		}
	}
	c.JSON(200, gin.H{"logs": service.PanelLogs(lines)})
	return nil
}

func panelEvents(c *gin.Context, d *Deps) error {
	list, err := service.PanelEvents(d.St, 200)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"events": list})
	return nil
}

func panelTestEmail(c *gin.Context, d *Deps) error {
	if err := service.TestEmail(d.St); err != nil {
		return service.BadRequest("邮件发送失败: " + err.Error())
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func panelConfig(c *gin.Context, d *Deps) error {
	raw := service.ConfigRaw(d.St)
	c.Header("Content-Type", "application/json")
	c.Data(200, "application/json", raw)
	return nil
}

func panelBackup(c *gin.Context, d *Deps) error {
	path, err := service.BackupToTemp(d.St)
	if err != nil {
		return err
	}
	defer os.Remove(path)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	c.Header("Content-Type", "application/gzip")
	c.Header("Content-Disposition", `attachment; filename="docker-manager-backup.tar.gz"`)
	_, _ = io.Copy(c.Writer, f)
	return nil
}

func panelRestore(c *gin.Context, d *Deps) error {
	file, err := c.FormFile("file")
	if err != nil {
		return service.BadRequest("backup.fileRequired")
	}
	f, err := file.Open()
	if err != nil {
		return service.BadRequest("backup.fileRequired")
	}
	defer f.Close()
	if err := service.RestoreFromReader(d.St, f); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true, "needRestart": true})
	return nil
}

func panelRestart(c *gin.Context, d *Deps) error {
	c.JSON(200, gin.H{"ok": true})
	service.RestartPanel()
	return nil
}
