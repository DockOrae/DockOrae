package api

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func monitorHost(c *gin.Context, d *Deps) error {
	c.JSON(200, service.HostInfo(d.St))
	return nil
}

func monitorMonitor(c *gin.Context, d *Deps) error {
	c.JSON(200, service.MonitorSnapshot(d.St))
	return nil
}

func systemPublicIP(c *gin.Context, d *Deps) error {
	v4, v6 := service.PublicIPs()
	c.JSON(200, gin.H{"ipv4": v4, "ipv6": v6})
	return nil
}

func monitorRegistryMirrors(c *gin.Context, d *Deps) error {
	mirrors, path, exists := service.RegistryMirrors()
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
	if err := service.SaveRegistryMirrors(payload.Mirrors); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true, "needRestart": true})
	return nil
}

func monitorRestartDocker(c *gin.Context, d *Deps) error {
	if err := service.RestartDocker(); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func systemEventsWS(c *gin.Context, d *Deps) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch := d.St.Events.Subscribe()
	defer d.St.Events.Unsubscribe(ch)
	_ = conn.WriteMessage(1, []byte(`{"type":"connected"}`))
	// GO-004:复用 ticker 而非每循环新建 time.After(事件频繁时避免 timer 堆积)
	pingTicker := time.NewTicker(30 * time.Second)
	defer pingTicker.Stop()
	for {
		select {
		case m, ok := <-ch:
			if !ok {
				return nil
			}
			payload, _ := json.Marshal(map[string]any{"type": "event", "data": state.EventToValue(m)})
			if conn.WriteMessage(1, payload) != nil {
				return nil
			}
		case <-pingTicker.C:
			if conn.WriteMessage(8, nil) != nil { // PingMessage
				return nil
			}
		}
	}
}
