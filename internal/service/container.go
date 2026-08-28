package service

import (
	"context"
	"encoding/json"
	"io"
	"net/netip"
	"os"
	"strconv"
	"strings"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/docker"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ContainersList 容器列表(精简字段)
func ContainersList(st *state.AppState, ctx context.Context, all bool) ([]model.ContainerListItem, error) {
	items, err := docker.ListContainers(st.Docker, ctx, client.ContainerListOptions{All: all})
	if err != nil {
		return nil, err
	}
	return filterManagedContainers(st, model.ToContainerItems(items)), nil
}

// filterManagedContainers 只保留面板管理的容器(1Panel 同款:外部创建的隐藏):
// 1) 有 createdBy 标签(面板创建 / 应用商店安装,如 createdBy=Apps)
// 2) compose 项目在面板数据目录(面板接管 / 面板编排的栈)
func filterManagedContainers(st *state.AppState, items []model.ContainerListItem) []model.ContainerListItem {
	kept := items[:0]
	for _, c := range items {
		if _, ok := c.Labels["createdBy"]; ok {
			kept = append(kept, c)
			continue
		}
		if proj := c.Labels["com.docker.compose.project"]; proj != "" {
			if _, err := os.Stat(ComposeFile(st, proj)); err == nil {
				kept = append(kept, c)
				continue
			}
		}
	}
	return kept
}

// ContainerInspect 容器详情(原始 JSON 由 handler 透传)
func ContainerInspect(st *state.AppState, ctx context.Context, id string) (container.InspectResponse, error) {
	return docker.InspectContainer(st.Docker, ctx, id)
}

// ContainerCreate 创建容器(许可证限制 + 参数组装)
func ContainerCreate(st *state.AppState, ctx context.Context, req model.CreateContainerReq) (string, error) {
	// 许可证限制:未激活时禁止创建容器(1Panel 商业版功能锁定)
	if !LicenseActive(st) {
		return "", NewApiError(403, "license.required")
	}
	if req.Image == "" {
		return "", BadRequest("container.imageEmpty")
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
			return "", BadRequest("container.mountSrcMissing")
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
		Labels:       map[string]string{"createdBy": "docker-manager"},
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

	return docker.CreateContainer(st.Docker, ctx, client.ContainerCreateOptions{
		Config:     cfg,
		HostConfig: hc,
		Name:       req.Name,
	})
}

// ContainerStart/Stop/Restart/Kill/Pause/Unpause/Rename/Remove 生命周期(1:1 透传)
func ContainerStart(st *state.AppState, ctx context.Context, id string) error {
	return docker.StartContainer(st.Docker, ctx, id)
}

func ContainerStop(st *state.AppState, ctx context.Context, id string, timeout *int) error {
	opts := client.ContainerStopOptions{}
	if timeout != nil {
		opts.Timeout = timeout
	}
	return docker.StopContainer(st.Docker, ctx, id, opts)
}

func ContainerRestart(st *state.AppState, ctx context.Context, id string, timeout *int) error {
	opts := client.ContainerRestartOptions{}
	if timeout != nil {
		opts.Timeout = timeout
	}
	return docker.RestartContainer(st.Docker, ctx, id, opts)
}

func ContainerKill(st *state.AppState, ctx context.Context, id string) error {
	return docker.KillContainer(st.Docker, ctx, id, "SIGKILL")
}

func ContainerPause(st *state.AppState, ctx context.Context, id string) error {
	return docker.PauseContainer(st.Docker, ctx, id)
}

func ContainerUnpause(st *state.AppState, ctx context.Context, id string) error {
	return docker.UnpauseContainer(st.Docker, ctx, id)
}

func ContainerRename(st *state.AppState, ctx context.Context, id, newName string) error {
	return docker.RenameContainer(st.Docker, ctx, id, newName)
}

func ContainerRemove(st *state.AppState, ctx context.Context, id string, force, removeVolumes bool) error {
	return docker.RemoveContainer(st.Docker, ctx, id, client.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: removeVolumes,
	})
}

func ContainersPrune(st *state.AppState, ctx context.Context) (container.PruneReport, error) {
	return docker.PruneContainers(st.Docker, ctx, client.ContainerPruneOptions{})
}

// ---------------- WebSocket 业务(日志/统计/终端) ----------------

// ContainerLogsStream 容器日志流(业务:构建日志选项;上层负责传输)
func ContainerLogsStream(st *state.AppState, ctx context.Context, id string, tail int64) (io.ReadCloser, error) {
	return docker.ContainerLogs(st.Docker, ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       strconv.FormatInt(tail, 10),
	})
}

// ContainerStatsPump 解码容器 stats 流并计算前端展示字段(cpu_pct/mem/net/pids),
// 每帧通过 emit 回调交给上层发送;返回 nil 表示流结束或被 emit 终止
func ContainerStatsPump(st *state.AppState, ctx context.Context, id string, emit func(payload map[string]any) bool) error {
	stats, err := docker.ContainerStats(st.Docker, ctx, id, client.ContainerStatsOptions{Stream: true})
	if err != nil {
		return err
	}
	defer stats.Body.Close()

	prev := [2]uint64{}
	hasPrev := false
	dec := json.NewDecoder(stats.Body)
	for {
		var s container.StatsResponse
		if dec.Decode(&s) != nil {
			return nil
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
		payload := map[string]any{
			"cpu_pct":   Round2(cpuPct),
			"mem_usage": memUsage,
			"mem_limit": memLimit,
			"mem_pct":   Round2(float64(memUsage) / float64(memLimit) * 100.0),
			"net_rx":    netRx,
			"net_tx":    netTx,
			"pids":      s.PidsStats.Current,
		}
		if !emit(payload) {
			return nil
		}
	}
}

// Round2 保留两位小数(monitor 速率与 stats 展示共用)
func Round2(v float64) float64 {
	return float64(int64(v*100+0.5)) / 100.0
}

// TerminalSession exec 终端会话:业务层持有 exec 生命周期,
// 上层只做 WS ↔ Reader/Stdin 的字节桥接与协议分发
type TerminalSession struct {
	Reader io.Reader
	Stdin  io.Writer
	execID string
	cli    *client.Client
	ctx    context.Context
	cancel context.CancelFunc
	close  func()
}

// CreateTerminal 创建 TTY exec 并 attach(返回会话,失败时资源已清理)
func CreateTerminal(st *state.AppState, ctx context.Context, id, shell string) (*TerminalSession, error) {
	execID, err := docker.ExecCreate(st.Docker, ctx, id, client.ExecCreateOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		Cmd:          []string{shell},
	})
	if err != nil {
		return nil, err
	}
	attach, err := docker.ExecAttach(st.Docker, ctx, execID, client.ExecAttachOptions{TTY: true})
	if err != nil {
		return nil, err
	}
	sctx, cancel := context.WithCancel(ctx)
	return &TerminalSession{
		Reader: attach.Reader,
		Stdin:  attach.Conn,
		execID: execID,
		cli:    st.Docker,
		ctx:    sctx,
		cancel: cancel,
		close:  attach.Close,
	}, nil
}

// Resize 调整 TTY 尺寸
func (s *TerminalSession) Resize(w, h int) error {
	return docker.ExecResize(s.cli, s.ctx, s.execID, client.ExecResizeOptions{Width: uint(w), Height: uint(h)})
}

// Close 终止会话(取消 ctx + 关闭 hijacked 连接)
func (s *TerminalSession) Close() {
	s.cancel()
	s.close()
}

// ContainerRecreate 重建容器(1Panel 同款):用原配置创建新容器 → 停删旧容器 → 启动新容器
func ContainerRecreate(st *state.AppState, ctx context.Context, id string) error {
	insp, err := docker.InspectContainer(st.Docker, ctx, id)
	if err != nil {
		return err
	}
	name := strings.TrimPrefix(insp.Name, "/")
	newID, err := docker.CreateContainer(st.Docker, ctx, client.ContainerCreateOptions{
		Name:             name,
		Config:           insp.Config,
		HostConfig:       insp.HostConfig,
		NetworkingConfig: &network.NetworkingConfig{EndpointsConfig: insp.NetworkSettings.Networks},
	})
	if err != nil {
		return err
	}
	_ = docker.StopContainer(st.Docker, ctx, id, client.ContainerStopOptions{})
	if err := docker.RemoveContainer(st.Docker, ctx, id, client.ContainerRemoveOptions{Force: true}); err != nil {
		return err
	}
	return docker.StartContainer(st.Docker, ctx, newID)
}
