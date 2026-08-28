package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/docker"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ContainerService 容器业务:依赖显式注入(docker client + license 检查 + compose 数据目录),
// 不直接依赖 AppState;license 与 composeDir 可注入,便于测试。
type ContainerService struct {
	docker  *client.Client
	license func() bool
	// composeDir 面板 compose 数据目录(用于"外部容器不显示"过滤)
	composeDir string
	// ops 容器重建所需的最小 Docker 操作集(测试可注入 fake)
	ops containerOps
}

// NewContainerService 生产构造:从 AppState 提取实际依赖
func NewContainerService(st *state.AppState) *ContainerService {
	return &ContainerService{
		docker:     st.Docker,
		license:    func() bool { return LicenseActive(st) },
		composeDir: st.ComposeDir,
		ops:        dockerContainerOps{cli: st.Docker},
	}
}

// containerOps 容器重建(Recreate)所需的最小 Docker 操作集:
// 仅包含实际使用的方法,供测试注入 fake,避免 mock 整个 Docker Client。
type containerOps interface {
	Inspect(ctx context.Context, id string) (container.InspectResponse, error)
	Create(ctx context.Context, opts client.ContainerCreateOptions) (string, error)
	Stop(ctx context.Context, id string, opts client.ContainerStopOptions) error
	Remove(ctx context.Context, id string, opts client.ContainerRemoveOptions) error
	Rename(ctx context.Context, id, newName string) error
	Start(ctx context.Context, id string) error
}

// dockerContainerOps 生产实现:包装 docker 包函数
type dockerContainerOps struct{ cli *client.Client }

func (d dockerContainerOps) Inspect(ctx context.Context, id string) (container.InspectResponse, error) {
	return docker.InspectContainer(d.cli, ctx, id)
}
func (d dockerContainerOps) Create(ctx context.Context, opts client.ContainerCreateOptions) (string, error) {
	return docker.CreateContainer(d.cli, ctx, opts)
}
func (d dockerContainerOps) Stop(ctx context.Context, id string, opts client.ContainerStopOptions) error {
	return docker.StopContainer(d.cli, ctx, id, opts)
}
func (d dockerContainerOps) Remove(ctx context.Context, id string, opts client.ContainerRemoveOptions) error {
	return docker.RemoveContainer(d.cli, ctx, id, opts)
}
func (d dockerContainerOps) Rename(ctx context.Context, id, newName string) error {
	return docker.RenameContainer(d.cli, ctx, id, newName)
}
func (d dockerContainerOps) Start(ctx context.Context, id string) error {
	return docker.StartContainer(d.cli, ctx, id)
}

// List 容器列表(精简字段,过滤外部容器)
func (s *ContainerService) List(ctx context.Context, all bool) ([]model.ContainerListItem, error) {
	items, err := docker.ListContainers(s.docker, ctx, client.ContainerListOptions{All: all})
	if err != nil {
		return nil, err
	}
	return s.managedFilter(model.ToContainerItems(items)), nil
}

// managedFilter 只保留面板管理的容器(1Panel 同款:外部创建的隐藏):
// 1) 有 createdBy 标签(面板创建 / 应用商店安装,如 createdBy=Apps)
// 2) compose 项目在面板数据目录(面板接管 / 面板编排的栈)
func (s *ContainerService) managedFilter(items []model.ContainerListItem) []model.ContainerListItem {
	kept := items[:0]
	for _, c := range items {
		if _, ok := c.Labels["createdBy"]; ok {
			kept = append(kept, c)
			continue
		}
		if proj := c.Labels["com.docker.compose.project"]; proj != "" {
			if _, err := os.Stat(composeFile(s.composeDir, proj)); err == nil {
				kept = append(kept, c)
				continue
			}
		}
	}
	return kept
}

// Inspect 容器详情(原始 JSON 由 handler 透传)
func (s *ContainerService) Inspect(ctx context.Context, id string) (container.InspectResponse, error) {
	return docker.InspectContainer(s.docker, ctx, id)
}

// Create 创建容器(许可证限制 + 参数转换)
func (s *ContainerService) Create(ctx context.Context, req model.CreateContainerReq) (string, error) {
	// 许可证限制:未激活时禁止创建容器(1Panel 商业版功能锁定)
	if !s.license() {
		return "", NewApiError(403, "license.required")
	}
	if req.Image == "" {
		return "", BadRequest("container.imageEmpty")
	}
	cfg, hc, err := buildContainerConfig(req)
	if err != nil {
		return "", err
	}
	return docker.CreateContainer(s.docker, ctx, client.ContainerCreateOptions{
		Config:     cfg,
		HostConfig: hc,
		Name:       req.Name,
	})
}

// buildContainerConfig 参数 → Docker Config/HostConfig(纯转换,无副作用,可单测)
func buildContainerConfig(req model.CreateContainerReq) (*container.Config, *container.HostConfig, error) {
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
			return nil, nil, BadRequest("container.mountSrcMissing")
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
	return cfg, hc, nil
}

// Start/Stop/Restart/Kill/Pause/Unpause/Rename/Remove/Prune 生命周期(1:1 透传)
func (s *ContainerService) Start(ctx context.Context, id string) error {
	return docker.StartContainer(s.docker, ctx, id)
}

func (s *ContainerService) Stop(ctx context.Context, id string, timeout *int) error {
	opts := client.ContainerStopOptions{}
	if timeout != nil {
		opts.Timeout = timeout
	}
	return docker.StopContainer(s.docker, ctx, id, opts)
}

func (s *ContainerService) Restart(ctx context.Context, id string, timeout *int) error {
	opts := client.ContainerRestartOptions{}
	if timeout != nil {
		opts.Timeout = timeout
	}
	return docker.RestartContainer(s.docker, ctx, id, opts)
}

func (s *ContainerService) Kill(ctx context.Context, id string) error {
	return docker.KillContainer(s.docker, ctx, id, "SIGKILL")
}

func (s *ContainerService) Pause(ctx context.Context, id string) error {
	return docker.PauseContainer(s.docker, ctx, id)
}

func (s *ContainerService) Unpause(ctx context.Context, id string) error {
	return docker.UnpauseContainer(s.docker, ctx, id)
}

func (s *ContainerService) Rename(ctx context.Context, id, newName string) error {
	return docker.RenameContainer(s.docker, ctx, id, newName)
}

func (s *ContainerService) Remove(ctx context.Context, id string, force, removeVolumes bool) error {
	return docker.RemoveContainer(s.docker, ctx, id, client.ContainerRemoveOptions{
		Force:         force,
		RemoveVolumes: removeVolumes,
	})
}

func (s *ContainerService) Prune(ctx context.Context) (container.PruneReport, error) {
	return docker.PruneContainers(s.docker, ctx, client.ContainerPruneOptions{})
}

// ---------------- WebSocket 业务(日志/统计/终端) ----------------

// LogsStream 容器日志流(业务:构建日志选项;上层负责传输)
func (s *ContainerService) LogsStream(ctx context.Context, id string, tail int64) (io.ReadCloser, error) {
	return docker.ContainerLogs(s.docker, ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Tail:       strconv.FormatInt(tail, 10),
	})
}

// StatsPump 解码容器 stats 流并计算前端展示字段(cpu_pct/mem/net/pids),
// 每帧通过 emit 回调交给上层发送;返回 nil 表示流结束或被 emit 终止
func (s *ContainerService) StatsPump(ctx context.Context, id string, emit func(payload map[string]any) bool) error {
	stats, err := docker.ContainerStats(s.docker, ctx, id, client.ContainerStatsOptions{Stream: true})
	if err != nil {
		return err
	}
	defer stats.Body.Close()

	prev := [2]uint64{}
	hasPrev := false
	dec := json.NewDecoder(stats.Body)
	for {
		var st container.StatsResponse
		err := dec.Decode(&st)
		if err != nil {
			// io.EOF = 流正常结束;其余解码错误要上报(如 Docker 返回错误帧)
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
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
			"cpu_pct":   Round2(cpuPct),
			"mem_usage": memUsage,
			"mem_limit": memLimit,
			"mem_pct":   Round2(float64(memUsage) / float64(memLimit) * 100.0),
			"net_rx":    netRx,
			"net_tx":    netTx,
			"pids":      st.PidsStats.Current,
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
	once   sync.Once
}

// CreateTerminal 创建 TTY exec 并 attach(返回会话,失败时资源已清理)
func (s *ContainerService) CreateTerminal(ctx context.Context, id, shell string) (*TerminalSession, error) {
	execID, err := docker.ExecCreate(s.docker, ctx, id, client.ExecCreateOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		TTY:          true,
		Cmd:          []string{shell},
	})
	if err != nil {
		return nil, err
	}
	attach, err := docker.ExecAttach(s.docker, ctx, execID, client.ExecAttachOptions{TTY: true})
	if err != nil {
		return nil, err
	}
	sctx, cancel := context.WithCancel(ctx)
	return &TerminalSession{
		Reader: attach.Reader,
		Stdin:  attach.Conn,
		execID: execID,
		cli:    s.docker,
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
	// 幂等:WebSocket 断开 / 输出 goroutine 退出 / context 取消等多路径可能同时触发
	s.once.Do(func() {
		s.cancel()
		s.close()
	})
}

// Recreate 重建容器:用原配置以临时名创建新容器 → 停删旧容器 → 临时名改回原名 → 启动。
// 修复:旧容器占用原名时同名创建必然 409;现用 <name>-recreate-<n> 临时名创建,
// 任一步失败回滚删除临时容器,旧容器不受影响(先建后删,数据卷安全)。
func (s *ContainerService) Recreate(ctx context.Context, id string) error {
	insp, err := s.ops.Inspect(ctx, id)
	if err != nil {
		return err
	}
	oldName := strings.TrimPrefix(insp.Name, "/")
	tmpName := fmt.Sprintf("%s-recreate-%d", oldName, time.Now().UnixNano()%100000)

	newID, err := s.ops.Create(ctx, client.ContainerCreateOptions{
		Name:             tmpName,
		Config:           insp.Config,
		HostConfig:       insp.HostConfig,
		NetworkingConfig: &network.NetworkingConfig{EndpointsConfig: insp.NetworkSettings.Networks},
	})
	if err != nil {
		return err
	}
	rollback := func() {
		_ = s.ops.Remove(ctx, newID, client.ContainerRemoveOptions{Force: true})
	}

	// 停旧容器:失败不中止(后续 Force 删除兜底,保持原语义)
	_ = s.ops.Stop(ctx, id, client.ContainerStopOptions{})
	if err := s.ops.Remove(ctx, id, client.ContainerRemoveOptions{Force: true}); err != nil {
		rollback()
		return err
	}
	// 临时名改回原名(原名已释放)
	if err := s.ops.Rename(ctx, newID, oldName); err != nil {
		// 改名失败(如原名被外部占用):旧容器已删,先启动临时名容器保证可用
		_ = s.ops.Start(ctx, newID)
		return err
	}
	return s.ops.Start(ctx, newID)
}
