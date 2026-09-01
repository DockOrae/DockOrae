package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"strconv"
	"strings"
	"time"

	"github.com/DockOrae/DockOrae/internal/agent"
	"github.com/DockOrae/DockOrae/internal/model"
	"github.com/DockOrae/DockOrae/internal/state"
)

// ContainerService 容器业务(§7:执行全部在 Agent,面板只管业务逻辑/权限/数据转换)。
type ContainerService struct {
	agent   *agent.Client
	license func() bool
	// ops 容器重建所需的最小 Agent 操作集(测试可注入 fake)
	ops recreateOps
}

// NewContainerService 生产构造:从 AppState 提取实际依赖
func NewContainerService(st *state.AppState) *ContainerService {
	return &ContainerService{
		agent:   st.Agent,
		license: func() bool { return LicenseFeatureActive(st, "container_create") },
		ops:     agentRecreateOps{client: st.Agent},
	}
}

// recreateOps Recreate 所需的最小 Agent 操作集
type recreateOps interface {
	ContainerInspectRaw(ctx context.Context, id string) (json.RawMessage, error)
	ContainerCreate(ctx context.Context, req agent.ContainerCreateReq) (string, error)
	ContainerStop(ctx context.Context, id string, timeout *int) error
	ContainerRemove(ctx context.Context, id string, force, removeVolumes bool) error
	ContainerRename(ctx context.Context, id, newName string) error
	ContainerStart(ctx context.Context, id string) error
}

// agentRecreateOps 生产实现:直接调 Agent 客户端
type agentRecreateOps struct{ client *agent.Client }

func (a agentRecreateOps) ContainerInspectRaw(ctx context.Context, id string) (json.RawMessage, error) {
	if a.client == nil {
		return nil, agentUnavailable()
	}
	return a.client.ContainerInspectRaw(ctx, id)
}
func (a agentRecreateOps) ContainerCreate(ctx context.Context, req agent.ContainerCreateReq) (string, error) {
	if a.client == nil {
		return "", agentUnavailable()
	}
	return a.client.ContainerCreate(ctx, req)
}
func (a agentRecreateOps) ContainerStop(ctx context.Context, id string, timeout *int) error {
	if a.client == nil {
		return agentUnavailable()
	}
	return a.client.ContainerStop(ctx, id, timeout)
}
func (a agentRecreateOps) ContainerRemove(ctx context.Context, id string, force, removeVolumes bool) error {
	if a.client == nil {
		return agentUnavailable()
	}
	return a.client.ContainerRemove(ctx, id, force, removeVolumes)
}
func (a agentRecreateOps) ContainerRename(ctx context.Context, id, newName string) error {
	if a.client == nil {
		return agentUnavailable()
	}
	return a.client.ContainerRename(ctx, id, newName)
}
func (a agentRecreateOps) ContainerStart(ctx context.Context, id string) error {
	if a.client == nil {
		return agentUnavailable()
	}
	return a.client.ContainerStart(ctx, id)
}

// ---------- Docker create 原始 JSON 结构(docker API 兼容,不依赖 moby) ----------

type containerConfigDocker struct {
	Image        string              `json:"Image"`
	Cmd          []string            `json:"Cmd,omitempty"`
	Env          []string            `json:"Env,omitempty"`
	Tty          bool                `json:"Tty"`
	AttachStdin  bool                `json:"AttachStdin"`
	OpenStdin    bool                `json:"OpenStdin"`
	ExposedPorts map[string]struct{} `json:"ExposedPorts,omitempty"`
	Labels       map[string]string   `json:"Labels,omitempty"`
}

type hostConfigDocker struct {
	PortBindings  map[string][]portBindingDocker `json:"PortBindings,omitempty"`
	Binds         []string                       `json:"Binds,omitempty"`
	RestartPolicy restartPolicyDocker            `json:"RestartPolicy"`
	Privileged    bool                           `json:"Privileged"`
	NetworkMode   string                         `json:"NetworkMode,omitempty"`
}

type portBindingDocker struct {
	HostIP   string `json:"HostIp,omitempty"`
	HostPort string `json:"HostPort"`
}

type restartPolicyDocker struct {
	Name string `json:"Name"`
}

// List 容器列表(精简字段;2026-09-02 起显示全部容器 —— 外部 docker run /
// docker compose up 的容器按 compose 标签实时分组展示,不再按面板管理范围过滤)
func (s *ContainerService) List(ctx context.Context, all bool) ([]model.ContainerListItem, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	items, err := s.agent.ContainerList(ctx, all)
	if err != nil {
		return nil, err
	}
	return model.ToContainerItems(items), nil
}

// Inspect 容器详情(原始 JSON 由 handler 透传)
func (s *ContainerService) Inspect(ctx context.Context, id string) (json.RawMessage, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	return s.agent.ContainerInspectRaw(ctx, id)
}

// Create 创建容器(许可证限制 + 参数转换 → Agent 执行)
func (s *ContainerService) Create(ctx context.Context, req model.CreateContainerReq) (string, error) {
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
	if s.agent == nil {
		return "", agentUnavailable()
	}
	cfgRaw, _ := json.Marshal(cfg)
	hcRaw, _ := json.Marshal(hc)
	return s.agent.ContainerCreate(ctx, agent.ContainerCreateReq{
		Name:       req.Name,
		Config:     cfgRaw,
		HostConfig: hcRaw,
	})
}

// buildContainerConfig 参数 → docker create 原始 JSON(纯转换,可单测)
func buildContainerConfig(req model.CreateContainerReq) (*containerConfigDocker, *hostConfigDocker, error) {
	exposed := map[string]struct{}{}
	bindings := map[string][]portBindingDocker{}
	for _, p := range req.Ports {
		portStr := p.Container
		if !strings.Contains(portStr, "/") {
			portStr += "/tcp"
		}
		port, err := parseDockerPort(portStr)
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
		bindings[port] = []portBindingDocker{
			{HostIP: hostIP, HostPort: strconv.Itoa(int(p.Host))},
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

	restart := "no"
	switch {
	case req.RestartPolicy != nil && *req.RestartPolicy == "always":
		restart = "always"
	case req.RestartPolicy != nil && *req.RestartPolicy == "unless-stopped":
		restart = "unless-stopped"
	case req.RestartPolicy != nil && *req.RestartPolicy == "on-failure":
		restart = "on-failure"
	}

	tty := req.Tty != nil && *req.Tty
	cfg := &containerConfigDocker{
		Image:        req.Image,
		Cmd:          req.Cmd,
		Env:          req.Env,
		Tty:          tty,
		AttachStdin:  tty,
		OpenStdin:    tty,
		ExposedPorts: exposed,
		Labels:       map[string]string{"createdBy": "docker-manager"},
	}
	hc := &hostConfigDocker{
		PortBindings:  bindings,
		Binds:         binds,
		RestartPolicy: restartPolicyDocker{Name: restart},
		Privileged:    req.Privileged != nil && *req.Privileged,
	}
	if req.Network != nil && *req.Network != "" {
		hc.NetworkMode = *req.Network
	}
	return cfg, hc, nil
}

// parseDockerPort 解析 "80" / "443/tcp" / "53/udp" → "80/tcp" 形式
func parseDockerPort(s string) (string, error) {
	parts := strings.SplitN(s, "/", 2)
	num, err := strconv.Atoi(parts[0])
	if err != nil || num < 1 || num > 65535 {
		return "", fmt.Errorf("invalid port %q", s)
	}
	proto := "tcp"
	if len(parts) == 2 && parts[1] != "" {
		proto = strings.ToLower(parts[1])
	}
	return strconv.Itoa(num) + "/" + proto, nil
}

// ---------- 生命周期(1:1 透传 Agent) ----------

func (s *ContainerService) Start(ctx context.Context, id string) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.ContainerStart(ctx, id)
}

func (s *ContainerService) Stop(ctx context.Context, id string, timeout *int) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.ContainerStop(ctx, id, timeout)
}

func (s *ContainerService) Restart(ctx context.Context, id string, timeout *int) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.ContainerRestart(ctx, id, timeout)
}

func (s *ContainerService) Kill(ctx context.Context, id string) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.ContainerKill(ctx, id)
}

func (s *ContainerService) Pause(ctx context.Context, id string) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.ContainerPause(ctx, id)
}

func (s *ContainerService) Unpause(ctx context.Context, id string) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.ContainerUnpause(ctx, id)
}

func (s *ContainerService) Rename(ctx context.Context, id, newName string) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.ContainerRename(ctx, id, newName)
}

func (s *ContainerService) Remove(ctx context.Context, id string, force, removeVolumes bool) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.ContainerRemove(ctx, id, force, removeVolumes)
}

func (s *ContainerService) Prune(ctx context.Context) (agent.PruneReport, error) {
	if s.agent == nil {
		return agent.PruneReport{}, agentUnavailable()
	}
	return s.agent.ContainerPrune(ctx)
}

// ---------- 容器命令执行(2026-09-02 容器终端 Exec 重构) ----------

// Exec 在容器内执行单条命令:浏览器不接触 Docker,全部经 Agent。
// 命令退出码非 0 不视为错误(HTTP 200 + exit_code 字段,由前端展示)。
// timeoutSeconds:1~300(0 = Agent 默认 30s)。
func (s *ContainerService) Exec(ctx context.Context, id, command string, timeoutSeconds int) (agent.ContainerExecResult, error) {
	if s.agent == nil {
		return agent.ContainerExecResult{}, agentUnavailable()
	}
	if timeoutSeconds != 0 && (timeoutSeconds < 1 || timeoutSeconds > 300) {
		return agent.ContainerExecResult{}, BadRequest("terminal.timeoutRange")
	}
	if command == "" {
		return agent.ContainerExecResult{}, BadRequest("terminal.commandEmpty")
	}
	res, err := s.agent.ContainerExec(ctx, id, command, timeoutSeconds)
	if err != nil {
		var ae *agent.AgentError
		if errors.As(err, &ae) {
			// Agent 明确错误(容器未运行/超时/并发上限/离线)→ 转对应 HTTP 状态
			return res, NewApiError(ae.Status, ae.Message)
		}
		return res, err
	}
	return res, nil
}

// ---------- 重建容器(业务逻辑在面板,底层原语走 Agent) ----------

// inspectShape 容器 inspect 原始 JSON 中重建所需字段
type inspectShape struct {
	Name   string `json:"Name"`
	Config struct {
		Image  string            `json:"Image"`
		Cmd    []string          `json:"Cmd"`
		Env    []string          `json:"Env"`
		Tty    bool              `json:"Tty"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	HostConfig      json.RawMessage `json:"HostConfig"`
	NetworkSettings struct {
		Networks json.RawMessage `json:"Networks"`
	} `json:"NetworkSettings"`
}

// Recreate 重建容器:用原配置以临时名创建新容器 → 停删旧容器 → 临时名改回原名 → 启动。
// 任一步失败回滚删除临时容器,旧容器不受影响(先建后删,数据卷安全)。
func (s *ContainerService) Recreate(ctx context.Context, id string) error {
	if s.ops == nil {
		return agentUnavailable()
	}
	raw, err := s.ops.ContainerInspectRaw(ctx, id)
	if err != nil {
		return err
	}
	var insp inspectShape
	if err := json.Unmarshal(raw, &insp); err != nil {
		return err
	}
	oldName := strings.TrimPrefix(insp.Name, "/")
	tmpName := fmt.Sprintf("%s-recreate-%d", oldName, time.Now().UnixNano()%100000)

	// 重建配置:Config 原样 + HostConfig 原样 + 原网络接入
	cfgRaw, _ := json.Marshal(map[string]any{
		"Image":       insp.Config.Image,
		"Cmd":         insp.Config.Cmd,
		"Env":         insp.Config.Env,
		"Tty":         insp.Config.Tty,
		"AttachStdin": insp.Config.Tty,
		"OpenStdin":   insp.Config.Tty,
		"Labels":      insp.Config.Labels,
	})
	networking := map[string]any{}
	if len(insp.NetworkSettings.Networks) > 0 {
		networking = map[string]any{"EndpointsConfig": insp.NetworkSettings.Networks}
	}
	ncRaw, _ := json.Marshal(networking)

	newID, err := s.ops.ContainerCreate(ctx, agent.ContainerCreateReq{
		Name:             tmpName,
		Config:           cfgRaw,
		HostConfig:       insp.HostConfig,
		NetworkingConfig: ncRaw,
	})
	if err != nil {
		return err
	}
	rollback := func() {
		_ = s.ops.ContainerRemove(ctx, newID, true, false)
	}

	// 停旧容器:失败不中止(后续 Force 删除兜底)
	_ = s.ops.ContainerStop(ctx, id, nil)
	if err := s.ops.ContainerRemove(ctx, id, true, false); err != nil {
		rollback()
		return err
	}
	// 临时名改回原名(原名已释放)
	if err := s.ops.ContainerRename(ctx, newID, oldName); err != nil {
		_ = s.ops.ContainerStart(ctx, newID)
		return err
	}
	return s.ops.ContainerStart(ctx, newID)
}

// agentUnavailable Agent 未部署/未配置时的统一错误
func agentUnavailable() error {
	return NewApiError(502, "agent.unavailable")
}
