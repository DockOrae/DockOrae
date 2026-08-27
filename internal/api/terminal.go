package api

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ---------------- 快速命令 ----------------

type QuickCommand struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Command string `json:"command"`
}

func dataFile(st *state.AppState, name string) string {
	return filepath.Join(st.Cfg.DataDir, name)
}

func loadCommands(path string) []QuickCommand {
	var cmds []QuickCommand
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &cmds)
	}
	return cmds
}

func saveCommands(path string, cmds []QuickCommand) error {
	out, err := json.MarshalIndent(cmds, "", "  ")
	if err != nil {
		return BadRequest(err.Error())
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return BadRequest("write failed: " + err.Error())
	}
	return nil
}

func terminalQuickCommands(c *gin.Context, st *state.AppState) error {
	path := dataFile(st, "quick_commands.json")
	c.JSON(200, gin.H{"commands": loadCommands(path)})
	return nil
}

func terminalAddQuickCommand(c *gin.Context, st *state.AppState) error {
	var payload QuickCommand
	if err := c.ShouldBindJSON(&payload); err != nil {
		return BadRequest("err.requestFailed")
	}
	if strings.TrimSpace(payload.Name) == "" || strings.TrimSpace(payload.Command) == "" {
		return BadRequest("terminal.nameAndCmdRequired")
	}
	path := dataFile(st, "quick_commands.json")
	cmds := loadCommands(path)
	cmd := QuickCommand{
		ID:      "c" + strconv.FormatInt(time.Now().UnixMilli(), 10),
		Name:    strings.TrimSpace(payload.Name),
		Command: strings.TrimSpace(payload.Command),
	}
	cmds = append(cmds, cmd)
	if err := saveCommands(path, cmds); err != nil {
		return err
	}
	c.JSON(200, gin.H{"command": cmd})
	return nil
}

func terminalDeleteQuickCommand(c *gin.Context, st *state.AppState) error {
	path := dataFile(st, "quick_commands.json")
	id := c.Param("id")
	var kept []QuickCommand
	for _, cmd := range loadCommands(path) {
		if cmd.ID != id {
			kept = append(kept, cmd)
		}
	}
	if err := saveCommands(path, kept); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// ---------------- 终端配置 ----------------

type TerminalSettings struct {
	FontFamily   string `json:"font_family"`
	FontSize     uint32 `json:"font_size"`
	Background   string `json:"background"`
	Foreground   string `json:"foreground"`
	CursorBlink  bool   `json:"cursor_blink"`
	Scrollback   uint32 `json:"scrollback"`
	DefaultShell string `json:"default_shell"`
}

func defaultSettings() TerminalSettings {
	return TerminalSettings{
		FontFamily:   "JetBrains Mono, Consolas, 'Courier New', monospace",
		FontSize:     13,
		Background:   "#0a0d13",
		Foreground:   "#e5e7eb",
		CursorBlink:  true,
		Scrollback:   2000,
		DefaultShell: "/bin/sh",
	}
}

func loadSettings(path string) TerminalSettings {
	s := defaultSettings()
	if raw, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(raw, &s)
	}
	return s
}

func terminalGetSettings(c *gin.Context, st *state.AppState) error {
	c.JSON(200, loadSettings(dataFile(st, "terminal.json")))
	return nil
}

func terminalSaveSettings(c *gin.Context, st *state.AppState) error {
	var payload TerminalSettings
	if err := c.ShouldBindJSON(&payload); err != nil {
		return BadRequest("err.requestFailed")
	}
	def := defaultSettings()
	s := TerminalSettings{
		FontFamily:   def.FontFamily,
		FontSize:     13,
		Background:   "#0a0d13",
		Foreground:   "#e5e7eb",
		DefaultShell: "/bin/sh",
	}
	if strings.TrimSpace(payload.FontFamily) != "" {
		s.FontFamily = payload.FontFamily
	}
	if payload.FontSize < 10 {
		s.FontSize = 10
	} else if payload.FontSize > 24 {
		s.FontSize = 24
	} else {
		s.FontSize = payload.FontSize
	}
	if payload.Background != "" {
		s.Background = payload.Background
	}
	if payload.Foreground != "" {
		s.Foreground = payload.Foreground
	}
	s.CursorBlink = payload.CursorBlink
	if payload.Scrollback < 500 {
		s.Scrollback = 500
	} else if payload.Scrollback > 10000 {
		s.Scrollback = 10000
	} else {
		s.Scrollback = payload.Scrollback
	}
	if payload.DefaultShell != "" {
		s.DefaultShell = payload.DefaultShell
	}
	out, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return BadRequest(err.Error())
	}
	if err := os.WriteFile(dataFile(st, "terminal.json"), out, 0o644); err != nil {
		return BadRequest("write failed: " + err.Error())
	}
	c.JSON(200, gin.H{"ok": true, "settings": s})
	return nil
}

// ---------------- 自身容器终端 ----------------

// selfContainerID 从 cgroup 解析面板自身容器 ID
func selfContainerID() (string, bool) {
	raw, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return "", false
	}
	for _, line := range strings.Split(string(raw), "\n") {
		parts := strings.Split(line, "/")
		last := parts[len(parts)-1]
		id := strings.TrimSuffix(strings.TrimPrefix(last, "docker-"), ".scope")
		if len(id) >= 12 && isHex(id) {
			return id, true
		}
	}
	return "", false
}

func isHex(s string) bool {
	for _, r := range s {
		if (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F') {
			continue
		}
		return false
	}
	return true
}

func terminalSelfContainer(c *gin.Context, st *state.AppState) error {
	id, ok := selfContainerID()
	if !ok {
		return NewApiError(400, "terminal.selfNotFound")
	}
	res, err := st.Docker.ContainerList(c.Request.Context(), client.ContainerListOptions{})
	if err != nil {
		return dockerError(err)
	}
	for _, ctr := range res.Items {
		if strings.HasPrefix(ctr.ID, id) || strings.HasPrefix(id, ctr.ID) {
			name := ""
			if len(ctr.Names) > 0 {
				name = ctr.Names[0]
			}
			c.JSON(200, gin.H{"id": ctr.ID, "name": name, "image": ctr.Image})
			return nil
		}
	}
	// 没在运行列表找到(自身容器可能没被 docker 看到)→ 返回 id 让前端尝试
	c.JSON(200, gin.H{"id": id, "name": "", "image": ""})
	return nil
}

// terminalSelfWS 自身容器 Web 终端;挂载了宿主机 rootfs(/host)时自动 chroot 进入宿主机
func terminalSelfWS(c *gin.Context, st *state.AppState) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	id, ok := selfContainerID()
	if !ok {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[self container not found]\r\n"))
		conn.Close()
		return nil
	}
	shell := c.Query("shell")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := "if [ -d /host/etc ]; then chroot /host " + shell + "; else " + shell + "; fi"
	if err := execCmdTerminalBridge(c, st, conn, id, cmd); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[exec failed: "+err.Error()+"]\r\n"))
		conn.Close()
		return nil
	}
	return nil
}

// execCmdTerminalBridge 用自定义 cmd 创建 exec 并桥接(自终端 chroot 用)
func execCmdTerminalBridge(c *gin.Context, st *state.AppState, conn *websocket.Conn, containerID, cmd string) error {
	execRes, err := st.Docker.ExecCreate(c.Request.Context(), containerID, client.ExecCreateOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		Cmd:          []string{"sh", "-c", cmd},
	})
	if err != nil {
		return err
	}
	attachRes, err := st.Docker.ExecAttach(c.Request.Context(), execRes.ID, client.ExecAttachOptions{TTY: true})
	if err != nil {
		return err
	}
	defer attachRes.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()

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
						_, _ = st.Docker.ExecResize(c.Request.Context(), execRes.ID, client.ExecResizeOptions{
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
