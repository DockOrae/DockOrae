package api

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func monitorHost(c *gin.Context, st *state.AppState) error {
	c.JSON(200, service.HostInfo(st))
	return nil
}

func monitorMonitor(c *gin.Context, st *state.AppState) error {
	c.JSON(200, service.MonitorSnapshot(st))
	return nil
}

func systemPublicIP(c *gin.Context, st *state.AppState) error {
	v4, v6 := service.PublicIPs()
	c.JSON(200, gin.H{"ipv4": v4, "ipv6": v6})
	return nil
}

func monitorRegistryMirrors(c *gin.Context, st *state.AppState) error {
	mirrors, path, exists := service.RegistryMirrors()
	c.JSON(200, gin.H{"mirrors": mirrors, "path": path, "exists": exists})
	return nil
}

func monitorSaveRegistryMirrors(c *gin.Context, st *state.AppState) error {
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

func monitorRestartDocker(c *gin.Context, st *state.AppState) error {
	if err := service.RestartDocker(); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func systemEventsWS(c *gin.Context, st *state.AppState) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()
	ch := st.Events.Subscribe()
	defer st.Events.Unsubscribe(ch)
	_ = conn.WriteMessage(1, []byte(`{"type":"connected"}`))
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
		case <-time.After(30 * time.Second):
			if conn.WriteMessage(8, nil) != nil { // PingMessage
				return nil
			}
		}
	}
}
