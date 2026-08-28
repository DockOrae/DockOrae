package api

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/moby/moby/api/pkg/stdcopy"

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
// 只做:升级连接、解析 query 参数、调 service 业务、字节桥接;不直接接触 moby

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
	logs, err := service.ContainerLogsStream(st, ctx, c.Param("id"), tail)
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
	// 解码与字段计算在 service 层,回调仅负责发送
	go func() {
		_ = service.ContainerStatsPump(st, ctx, c.Param("id"), func(payload map[string]any) bool {
			raw, _ := json.Marshal(payload)
			if conn.WriteMessage(websocket.TextMessage, raw) != nil {
				cancel()
				return false
			}
			return true
		})
		cancel()
	}()

	wsPump(ctx, conn, func(mt int, data []byte) bool {
		return mt != websocket.CloseMessage
	})
	_ = conn.WriteMessage(websocket.CloseMessage, nil)
	return nil
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
	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	session, err := service.CreateTerminal(st, ctx, c.Param("id"), shell)
	if err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[exec failed: "+err.Error()+"]\r\n"))
		return nil
	}
	defer session.Close()

	// exec 输出 → ws 二进制
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := session.Reader.Read(buf)
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
			if _, err := session.Stdin.Write(data); err != nil {
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
						_ = session.Resize(w, h)
					}
				}
			} else if text == "stop" {
				return false
			} else {
				_, _ = session.Stdin.Write(data)
			}
		case websocket.CloseMessage:
			return false
		}
		return true
	})
	_ = conn.WriteMessage(websocket.CloseMessage, nil)
	return nil
}
