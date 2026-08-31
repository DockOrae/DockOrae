package api

import (
	"context"
	"encoding/json"
	"strconv"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/DockOrae/DockOrae/internal/agent"
	"github.com/DockOrae/DockOrae/internal/model"
	"github.com/DockOrae/DockOrae/internal/service"
)

// ---------------- 列表 / 详情 / 生命周期 ----------------

func containersList(c *gin.Context, d *Deps) error {
	all := true
	if c.Query("all") != "" {
		all = parseBool(c.Query("all"), true)
	}
	items, err := d.Containers.List(c.Request.Context(), all)
	if err != nil {
		return err
	}
	c.JSON(200, items)
	return nil
}

func containersInspect(c *gin.Context, d *Deps) error {
	raw, err := d.Containers.Inspect(c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}
	c.Data(200, "application/json", raw)
	return nil
}

func containersCreate(c *gin.Context, d *Deps) error {
	var req model.CreateContainerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	id, err := d.Containers.Create(c.Request.Context(), req)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"id": id})
	return nil
}

func containersPrune(c *gin.Context, d *Deps) error {
	report, err := d.Containers.Prune(c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, report)
	return nil
}

func containersRemove(c *gin.Context, d *Deps) error {
	err := d.Containers.Remove(c.Request.Context(), c.Param("id"),
		parseBool(c.Query("force"), false), parseBool(c.Query("v"), false))
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// containersRecreate 重建容器(保留原配置)
func containersRecreate(c *gin.Context, d *Deps) error {
	if err := d.Containers.Recreate(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersStart(c *gin.Context, d *Deps) error {
	if err := d.Containers.Start(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersStop(c *gin.Context, d *Deps) error {
	err := d.Containers.Stop(c.Request.Context(), c.Param("id"), nil)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersRestart(c *gin.Context, d *Deps) error {
	err := d.Containers.Restart(c.Request.Context(), c.Param("id"), nil)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersKill(c *gin.Context, d *Deps) error {
	if err := d.Containers.Kill(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersPause(c *gin.Context, d *Deps) error {
	if err := d.Containers.Pause(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersUnpause(c *gin.Context, d *Deps) error {
	if err := d.Containers.Unpause(c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersRename(c *gin.Context, d *Deps) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.Containers.Rename(c.Request.Context(), c.Param("id"), req.Name); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// ---------------- WebSocket:日志 / 统计 / 终端 ----------------
// §5/§7:执行在 Agent,面板只做浏览器 ↔ Agent 的双向透传(统计字段计算留在面板)。

// relayWS 浏览器 WS ↔ Agent WS 全双工透传
func relayWS(c *gin.Context, conn *websocket.Conn, aconn *websocket.Conn) {
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	defer aconn.Close()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() { // Agent → 浏览器
		defer wg.Done()
		for {
			mt, data, err := aconn.ReadMessage()
			if err != nil {
				cancel()
				return
			}
			if conn.WriteMessage(mt, data) != nil {
				cancel()
				return
			}
		}
	}()
	wsPump(ctx, conn, func(mt int, data []byte) bool { // 浏览器 → Agent
		if mt == websocket.CloseMessage {
			return false
		}
		if aconn.WriteMessage(mt, data) != nil {
			return false
		}
		return true
	})
	cancel()
	// 浏览器已断开(wsPump 返回):先关闭 Agent 连接,解除对端 ReadMessage 的阻塞,
	// 再等待转发协程退出 —— 否则 Agent 流不结束(如 docker logs -f)时 wg.Wait 永久阻塞,
	// 协程 + Agent 连接随每次浏览器断开累积泄漏。
	aconn.Close()
	wg.Wait()
}

func containersLogsWS(c *gin.Context, d *Deps) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()

	tail := "500"
	if t := c.Query("tail"); t != "" {
		if _, err := strconv.ParseInt(t, 10, 64); err == nil {
			tail = t
		}
	}
	aconn, err := d.St.Agent.ContainerLogsWS(c.Request.Context(), c.Param("id"), tail)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		_ = conn.WriteMessage(websocket.CloseMessage, nil)
		return nil
	}
	relayWS(c, conn, aconn)
	return nil
}

func containersStatsWS(c *gin.Context, d *Deps) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	aconn, err := d.St.Agent.ContainerStatsWS(ctx, c.Param("id"))
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte(err.Error()))
		_ = conn.WriteMessage(websocket.CloseMessage, nil)
		return nil
	}
	defer aconn.Close()

	// Agent 原始帧 → 计算前端字段 → 浏览器(差分逻辑与原实现一致)
	prev := [2]uint64{}
	hasPrev := false
	for {
		_, data, err := aconn.ReadMessage()
		if err != nil {
			break
		}
		var st agent.StatsFrame
		if json.Unmarshal(data, &st) != nil {
			continue
		}
		cpuTotal := st.CPUStats.CPUUsage.TotalUsage
		sys := st.CPUStats.SystemUsage
		cpuPct := 0.0
		if hasPrev {
			d1 := cpuTotal - prev[0]
			d2 := sys - prev[1]
			if d2 > 0 {
				cpuPct = float64(d1) / float64(d2) * float64(st.CPUStats.OnlineCPUs) * 100.0
			}
		}
		prev = [2]uint64{cpuTotal, sys}
		hasPrev = true

		memUsage := st.MemoryStats.Usage
		memLimit := st.MemoryStats.Limit
		if memLimit < 1 {
			memLimit = 1
		}
		var netRx, netTx uint64
		for _, n := range st.Networks {
			netRx += n.RxBytes
			netTx += n.TxBytes
		}
		payload := map[string]any{
			"cpu_pct":   service.Round2(cpuPct),
			"mem_usage": memUsage,
			"mem_limit": memLimit,
			"mem_pct":   service.Round2(float64(memUsage) / float64(memLimit) * 100.0),
			"net_rx":    netRx,
			"net_tx":    netTx,
			"pids":      st.PidsStats.Current,
		}
		raw, _ := json.Marshal(payload)
		if conn.WriteMessage(websocket.TextMessage, raw) != nil {
			break
		}
	}
	_ = conn.WriteMessage(websocket.CloseMessage, nil)
	return nil
}

func containersTerminalWS(c *gin.Context, d *Deps) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()

	shell := "/bin/sh"
	if s := c.Query("shell"); s != "" {
		shell = s
	}
	aconn, err := d.St.Agent.ContainerTerminalWS(c.Request.Context(), c.Param("id"), shell)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[exec failed: "+err.Error()+"]\r\n"))
		return nil
	}
	relayWS(c, conn, aconn)
	return nil
}
