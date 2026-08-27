package api

import (
	"context"
	"encoding/json"
	"mime"
	"net/netip"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

func mimeByExt(name string) string {
	if t := mime.TypeByExtension(filepath.Ext(name)); t != "" {
		return t
	}
	return "application/octet-stream"
}

// ---------------- 列表 / 详情 ----------------

type containersListQuery struct {
	All *bool `form:"all"`
}

func containersList(c *gin.Context, st *state.AppState) error {
	all := true
	if c.Query("all") != "" {
		all = parseBool(c.Query("all"), true)
	}
	res, err := st.Docker.ContainerList(c.Request.Context(), client.ContainerListOptions{All: all})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, res.Items)
	return nil
}

func parseBool(s string, def bool) bool {
	if b, err := strconv.ParseBool(s); err == nil {
		return b
	}
	return def
}

func containersInspect(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.ContainerInspect(c.Request.Context(), c.Param("id"), client.ContainerInspectOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.Data(200, "application/json", res.Raw)
	return nil
}

// ---------------- 创建 ----------------

type portMap struct {
	Container string  `json:"container"`
	Host      uint16  `json:"host"`
	HostIP    *string `json:"host_ip"`
}

type volumeMap struct {
	Host      *string `json:"host"`
	Volume    *string `json:"volume"`
	Container string  `json:"container"`
	Mode      *string `json:"mode"`
}

type createReq struct {
	Name          string      `json:"name"`
	Image         string      `json:"image"`
	Cmd           []string    `json:"cmd"`
	Env           []string    `json:"env"`
	Ports         []portMap   `json:"ports"`
	Volumes       []volumeMap `json:"volumes"`
	Network       *string     `json:"network"`
	RestartPolicy *string     `json:"restart_policy"`
	Tty           *bool       `json:"tty"`
	Privileged    *bool       `json:"privileged"`
}

func containersCreate(c *gin.Context, st *state.AppState) error {
	// 许可证限制:未激活时禁止创建容器(1Panel 商业版功能锁定)
	if !licenseActive(st) {
		return NewApiError(403, "license.required")
	}
	var req createReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	if req.Image == "" {
		return BadRequest("container.imageEmpty")
	}

	exposed := network.PortSet{}
	bindings := network.PortMap{}
	for _, p := range req.Ports {
		portStr := p.Container
		if !strings.Contains(portStr, "/") {
			portStr += "/tcp"
		}
		port, err := network.ParsePort(portStr)
		if err != nil {
			continue
		}
		exposed[port] = struct{}{}
		hostIP := "0.0.0.0"
		if p.HostIP != nil && *p.HostIP != "" {
			if ip, err := netip.ParseAddr(*p.HostIP); err == nil {
				hostIP = ip.String()
			}
		}
		bindings[port] = []network.PortBinding{
			{HostIP: netip.MustParseAddr(hostIP), HostPort: strconv.Itoa(int(p.Host))},
		}
	}

	var binds []string
	for _, v := range req.Volumes {
		var src string
		switch {
		case v.Host != nil && *v.Host != "":
			src = *v.Host
		case v.Volume != nil && *v.Volume != "":
			src = *v.Volume
		default:
			return BadRequest("container.mountSrcMissing")
		}
		mode := "rw"
		if v.Mode != nil && *v.Mode != "" {
			mode = *v.Mode
		}
		binds = append(binds, src+":"+v.Container+":"+mode)
	}

	restart := container.RestartPolicyDisabled
	switch {
	case req.RestartPolicy != nil && *req.RestartPolicy == "always":
		restart = container.RestartPolicyAlways
	case req.RestartPolicy != nil && *req.RestartPolicy == "unless-stopped":
		restart = container.RestartPolicyUnlessStopped
	case req.RestartPolicy != nil && *req.RestartPolicy == "on-failure":
		restart = container.RestartPolicyOnFailure
	}

	tty := req.Tty != nil && *req.Tty
	cfg := &container.Config{
		Image:        req.Image,
		Cmd:          req.Cmd,
		Env:          req.Env,
		Tty:          tty,
		AttachStdin:  tty,
		OpenStdin:    tty,
		ExposedPorts: exposed,
	}
	hc := &container.HostConfig{
		PortBindings:  bindings,
		Binds:         binds,
		RestartPolicy: container.RestartPolicy{Name: restart},
		Privileged:    req.Privileged != nil && *req.Privileged,
	}
	if req.Network != nil && *req.Network != "" {
		hc.NetworkMode = container.NetworkMode(*req.Network)
	}

	res, err := st.Docker.ContainerCreate(c.Request.Context(), client.ContainerCreateOptions{
		Config:     cfg,
		HostConfig: hc,
		Name:       req.Name,
	})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"id": res.ID, "warnings": res.Warnings})
	return nil
}

// ---------------- 生命周期操作 ----------------

func containersStart(c *gin.Context, st *state.AppState) error {
	_, err := st.Docker.ContainerStart(c.Request.Context(), c.Param("id"), client.ContainerStartOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersStop(c *gin.Context, st *state.AppState) error {
	_, err := st.Docker.ContainerStop(c.Request.Context(), c.Param("id"), client.ContainerStopOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersRestart(c *gin.Context, st *state.AppState) error {
	_, err := st.Docker.ContainerRestart(c.Request.Context(), c.Param("id"), client.ContainerRestartOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersKill(c *gin.Context, st *state.AppState) error {
	_, err := st.Docker.ContainerKill(c.Request.Context(), c.Param("id"), client.ContainerKillOptions{Signal: "SIGKILL"})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersPause(c *gin.Context, st *state.AppState) error {
	_, err := st.Docker.ContainerPause(c.Request.Context(), c.Param("id"), client.ContainerPauseOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersUnpause(c *gin.Context, st *state.AppState) error {
	_, err := st.Docker.ContainerUnpause(c.Request.Context(), c.Param("id"), client.ContainerUnpauseOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

type renameReq struct {
	Name string `json:"name"`
}

func containersRename(c *gin.Context, st *state.AppState) error {
	var req renameReq
	if err := c.ShouldBindJSON(&req); err != nil {
		return BadRequest("err.requestFailed")
	}
	_, err := st.Docker.ContainerRename(c.Request.Context(), c.Param("id"), client.ContainerRenameOptions{NewName: req.Name})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

type removeQuery struct {
	Force   *bool `form:"force"`
	Volumes *bool `form:"volumes"`
}

func containersRemove(c *gin.Context, st *state.AppState) error {
	_, err := st.Docker.ContainerRemove(c.Request.Context(), c.Param("id"), client.ContainerRemoveOptions{
		Force:         parseBool(c.Query("force"), false),
		RemoveVolumes: parseBool(c.Query("volumes"), false),
	})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

func containersPrune(c *gin.Context, st *state.AppState) error {
	res, err := st.Docker.ContainerPrune(c.Request.Context(), client.ContainerPruneOptions{})
	if err != nil {
		return dockerError(err)
	}
	c.JSON(200, res.Report)
	return nil
}

// ---------------- 日志 (WebSocket 实时) ----------------

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
	logs, err := st.Docker.ContainerLogs(ctx, c.Param("id"), client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       strconv.FormatInt(tail, 10),
	})
	if err != nil {
		return dockerError(err)
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
	if err := w.conn.WriteMessage(websocket.TextMessage, p); err != nil {
		return 0, err
	}
	return len(p), nil
}

// ---------------- 统计 (WebSocket 实时) ----------------

func containersStatsWS(c *gin.Context, st *state.AppState) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	defer conn.Close()

	ctx, cancel := context.WithCancel(c.Request.Context())
	defer cancel()
	res, err := st.Docker.ContainerStats(ctx, c.Param("id"), client.ContainerStatsOptions{Stream: true})
	if err != nil {
		return dockerError(err)
	}
	defer res.Body.Close()

	prev := [2]uint64{}
	hasPrev := false
	go func() {
		dec := json.NewDecoder(res.Body)
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

// ---------------- Web 终端 (exec + WebSocket) ----------------

func containersTerminalWS(c *gin.Context, st *state.AppState) error {
	conn, err := upgradeWS(c)
	if err != nil {
		return err
	}
	shell := c.Query("shell")
	if shell == "" {
		shell = "/bin/sh"
	}
	if err := execTerminalBridge(c, st, conn, c.Param("id"), shell); err != nil {
		_ = conn.WriteMessage(websocket.TextMessage, []byte("[exec failed: "+err.Error()+"]\r\n"))
		conn.Close()
		return nil
	}
	return nil
}

// execTerminalBridge 创建 exec(TTY)并桥接 WS ↔ hijacked 连接
func execTerminalBridge(c *gin.Context, st *state.AppState, conn *websocket.Conn, containerID, shell string) error {
	execRes, err := st.Docker.ExecCreate(c.Request.Context(), containerID, client.ExecCreateOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		Cmd:          []string{shell},
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

	// ws → exec 输入 + resize
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
