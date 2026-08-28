package api

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/docker"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ---------------- 列表 / 详情 / 生命周期 ----------------

func containersList(c *gin.Context, st *state.AppState) error {
	all := true
	if c.Query("all") != "" {
		all = parseBool(c.Query("all"), true)
	}
	items, err := service.ContainersList(st, c.Request.Context(), all)
	if err != nil {
		return err
	}
	c.JSON(200, items)
	return nil
}

func containersInspect(c *gin.Context, st *state.AppState) error {
	insp, err := service.ContainerInspect(st, c.Request.Context(), c.Param("id"))
	if err != nil {
		return err
	}
	raw, _ := json.Marshal(insp)
	c.Data(200, "application/json", raw)
	return nil
}

func containersCreate(c *gin.Context, st *state.AppState) error {
	var req model.CreateContainerReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	id, err := service.ContainerCreate(st, c.Request.Context(), req)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"id": id})
	return nil
}

func containersPrune(c *gin.Context, st *state.AppState) error {
	report, err := service.ContainersPrune(st, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, report)
	return nil
}

func containersRemove(c *gin.Context, st *state.AppState) error {
	err := service.ContainerRemove(st, c.Request.Context(), c.Param("id"),
		parseBool(c.Query("force"), false), parseBool(c.Query("v"), false))
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersStart(c *gin.Context, st *state.AppState) error {
	if err := service.ContainerStart(st, c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersStop(c *gin.Context, st *state.AppState) error {
	err := service.ContainerStop(st, c.Request.Context(), c.Param("id"), nil)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersRestart(c *gin.Context, st *state.AppState) error {
	err := service.ContainerRestart(st, c.Request.Context(), c.Param("id"), nil)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersKill(c *gin.Context, st *state.AppState) error {
	if err := service.ContainerKill(st, c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersPause(c *gin.Context, st *state.AppState) error {
	if err := service.ContainerPause(st, c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersUnpause(c *gin.Context, st *state.AppState) error {
	if err := service.ContainerUnpause(st, c.Request.Context(), c.Param("id")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersRename(c *gin.Context, st *state.AppState) error {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Name == "" {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.ContainerRename(st, c.Request.Context(), c.Param("id"), req.Name); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// ---------------- WebSocket:日志 / 统计 / 终端 ----------------

func containersLogsWS(c *gin.Context, st *state.AppState) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()

	tail := int64(500)
	if t := c.Query("tail"); t != "" {
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			tail = n
		}
	}
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	logs, err := docker.ContainerLogs(st.Docker, ctx, c.Param("id"), client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       strconv.FormatInt(tail, 10),
	})
	if err != nil {
		return err
	}
	defer logs.Close()

	// 后台把 stdout/stderr 解复用后逐条发文本消息
	w := wsTextWriter{conn: conn}
	go func() {
		_, _ = stdcopy.StdCopy(w, w, logs)
		cancel()
	}()

	wsPump(ctx, conn, func(mt int, data []byte) bool {
		if mt == websocket.TextMessage && string(data) == "stop" {
			return false
		}
		if mt == websocket.CloseMessage {
			return false
		}
		return true
	})
	_ = conn.WriteMessage(websocket.CloseMessage, nil)
	return nil
}

type wsTextWriter struct {
	conn *websocket.Conn
}

func (w wsTextWriter) Write(p []byte) (int, error) {
	_ = w.conn.WriteMessage(websocket.TextMessage, p)
	return len(p), nil
}

func containersStatsWS(c *gin.Context, st *state.AppState) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	stats, err := docker.ContainerStats(st.Docker, ctx, c.Param("id"), client.ContainerStatsOptions{Stream: true})
	if err != nil {
		return err
	}
	defer stats.Body.Close()

	// 服务端解码 stats 流并计算前端展示字段(cpu_pct/mem/net/pids)
	prev := [2]uint64{}
	hasPrev := false
	go func() {
		dec := json.NewDecoder(stats.Body)
		for {
			var s container.StatsResponse
			if dec.Decode(&s) != nil {
				break
			}
			cpuTotal := s.CPUStats.CPUUsage.TotalUsage
			sys := s.CPUStats.SystemUsage
			cpuPct := 0.0
			if hasPrev {
				d1 := cpuTotal - prev[0]
				d2 := sys - prev[1]
				if d2 > 0 {
					cpuPct = float64(d1) / float64(d2) * float64(s.CPUStats.OnlineCPUs) * 100.0
				}
			}
			prev = [2]uint64{cpuTotal, sys}
			hasPrev = true

			memUsage := s.MemoryStats.Usage
			memLimit := s.MemoryStats.Limit
			if memLimit < 1 {
				memLimit = 1
			}
			var netRx, netTx uint64
			for _, n := range s.Networks {
				netRx += n.RxBytes
				netTx += n.TxBytes
			}
			payload, _ := json.Marshal(gin.H{
				"cpu_pct":   round2(cpuPct),
				"mem_usage": memUsage,
				"mem_limit": memLimit,
				"mem_pct":   round2(float64(memUsage) / float64(memLimit) * 100.0),
				"net_rx":    netRx,
				"net_tx":    netTx,
				"pids":      s.PidsStats.Current,
			})
			if conn.WriteMessage(websocket.TextMessage, payload) != nil {
				cancel()
				return
			}
		}
		cancel()
	}()

	wsPump(ctx, conn, func(mt int, data []byte) bool {
		return mt != websocket.CloseMessage
	})
	_ = conn.WriteMessage(websocket.CloseMessage, nil)
	return nil
}

func round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100.0
}

func containersTerminalWS(c *gin.Context, st *state.AppState) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()

	shell := "/bin/sh"
	if s := c.Query("shell"); s != "" {
		shell = s
	}
	if err := execTerminalBridge(c, st, conn, c.Param("id"), shell); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[exec failed: "+err.Error()+"]\r\n"))
		conn.Close()
	}
	return nil
}

// execTerminalBridge 创建 exec(TTY)并桥接 WS ↔ hijacked 连接
func execTerminalBridge(c *gin.Context, st *state.AppState, conn *websocket.Conn, containerID, shell string) error {
	execID, err := docker.ExecCreate(st.Docker, c.Request.Context(), containerID, client.ExecCreateOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		Cmd:          []string{shell},
	})
	if err != nil {
		return err
	}
	attachRes, err := docker.ExecAttach(st.Docker, c.Request.Context(), execID, client.ExecAttachOptions{TTY: true})
	if err != nil {
		return err
	}
	defer attachRes.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

	// exec 输出 → ws 二进制
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := attachRes.Reader.Read(buf)
			if n > 0 {
				if conn.WriteMessage(websocket.BinaryMessage, buf[:n]) != nil {
					cancel()
					return
				}
			}
			if err != nil {
				cancel()
				return
			}
		}
	}()

	// ws → exec 输入 + resize/stop 控制协议(单读方,避免并发读)
	wsPump(ctx, conn, func(mt int, data []byte) bool {
		switch mt {
		case websocket.BinaryMessage:
			if _, err := attachRes.Conn.Write(data); err != nil {
				return false
			}
		case websocket.TextMessage:
			text := string(data)
			if strings.HasPrefix(text, "resize:") {
				parts := strings.SplitN(strings.TrimPrefix(text, "resize:"), ",", 2)
				if len(parts) == 2 {
					w, err1 := strconv.Atoi(parts[0])
					h, err2 := strconv.Atoi(parts[1])
					if err1 == nil && err2 == nil {
						_ = docker.ExecResize(st.Docker, c.Request.Context(), execID, client.ExecResizeOptions{
							Height: uint(h),
							Width:  uint(w),
						})
					}
				}
			} else if text == "stop" {
				return false
			} else {
				_, _ = attachRes.Conn.Write(data)
			}
		case websocket.CloseMessage:
			return false
		}
		return true
	})
	_ = conn.WriteMessage(websocket.CloseMessage, nil)
	return nil
}
