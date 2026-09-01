package api

import (
	"github.com/gin-gonic/gin"

	"github.com/DockOrae/DockOrae/internal/service"
)

func monitorHost(c *gin.Context, d *Deps) error {
	c.JSON(200, service.HostInfo(d.St))
	return nil
}

func monitorMonitor(c *gin.Context, d *Deps) error {
	c.JSON(200, service.MonitorSnapshot(d.St))
	return nil
}

func monitorRegistryMirrors(c *gin.Context, d *Deps) error {
	mirrors, path, exists := service.RegistryMirrors(d.St)
	c.JSON(200, gin.H{"mirrors": mirrors, "path": path, "exists": exists})
	return nil
}

func monitorSaveRegistryMirrors(c *gin.Context, d *Deps) error {
	var payload struct {
		Mirrors []string `json:"mirrors"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.SaveRegistryMirrors(d.St, payload.Mirrors); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true, "needRestart": true})
	return nil
}
