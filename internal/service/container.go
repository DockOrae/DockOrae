package service

import (
	"context"
	"net/netip"
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
	return model.ToContainerItems(items), nil
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
