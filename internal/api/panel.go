package api

import (
	"io"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func panelSettings(c *gin.Context, st *state.AppState) error {
	c.JSON(200, service.SettingsGet(st))
	return nil
}

func panelSettingsSave(c *gin.Context, st *state.AppState) error {
	var patch map[string]any
	if err := c.ShouldBindJSON(&patch); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.SettingsUpdate(st, patch); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true, "needRestart": true})
	return nil
}

func panelLogs(c *gin.Context, st *state.AppState) error {
	lines := 500
	if v := c.Query("lines"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 5000 {
			lines = n
		}
	}
	c.JSON(200, gin.H{"logs": service.PanelLogs(lines)})
	return nil
}

func panelEvents(c *gin.Context, st *state.AppState) error {
	list, err := service.PanelEvents(st, 200)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"events": list})
	return nil
}

func panelTestEmail(c *gin.Context, st *state.AppState) error {
	if err := service.TestEmail(st); err != nil {
		return service.BadRequest("邮件发送失败: " + err.Error())
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func panelConfig(c *gin.Context, st *state.AppState) error {
	raw := service.ConfigRaw(st)
	c.Header("Content-Type", "application/json")
	c.Data(200, "application/json", raw)
	return nil
}

func panelBackup(c *gin.Context, st *state.AppState) error {
	path, err := service.BackupToTemp(st)
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

func panelRestore(c *gin.Context, st *state.AppState) error {
	file, err := c.FormFile("file")
	if err != nil {
		return service.BadRequest("backup.fileRequired")
	}
	f, err := file.Open()
	if err != nil {
		return service.BadRequest("backup.fileRequired")
	}
	defer f.Close()
	if err := service.RestoreFromReader(st, f); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true, "needRestart": true})
	return nil
}

func panelRestart(c *gin.Context, st *state.AppState) error {
	c.JSON(200, gin.H{"ok": true})
	service.RestartPanel()
	return nil
}
