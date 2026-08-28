package service

import (
	"context"
	"net/netip"

	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/docker"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// NetworksList 网络列表(精简字段)
func NetworksList(st *state.AppState, ctx context.Context) ([]model.NetworkListItem, error) {
	items, err := docker.ListNetworks(st.Docker, ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}
	return model.ToNetworkItems(items), nil
}

// NetworkInspect 网络详情(原始 JSON 透传)
func NetworkInspect(st *state.AppState, ctx context.Context, id string) (network.Inspect, error) {
	net, _, err := docker.InspectNetwork(st.Docker, ctx, id)
	return net, err
}

// NetworkRaw 网络详情原始 JSON(handler 透传)
func NetworkRaw(st *state.AppState, ctx context.Context, id string) ([]byte, error) {
	_, raw, err := docker.InspectNetwork(st.Docker, ctx, id)
	return raw, err
}

// NetworkCreate 创建网络
func NetworkCreate(st *state.AppState, ctx context.Context, req model.CreateNetworkReq) (string, error) {
	if req.Name == "" {
		return "", BadRequest("network.nameEmpty")
	}
	driver := "bridge"
	if req.Driver != nil && *req.Driver != "" {
		driver = *req.Driver
	}
	opts := client.NetworkCreateOptions{
		Driver:     driver,
		Internal:   req.Internal != nil && *req.Internal,
		EnableIPv6: Ptr(false),
	}
	if req.Subnet != nil || req.Gateway != nil {
		ipamCfg := network.IPAMConfig{}
		if req.Subnet != nil {
			if p, err := netip.ParsePrefix(*req.Subnet); err == nil {
				ipamCfg.Subnet = p
			}
		}
		if req.Gateway != nil {
			if a, err := netip.ParseAddr(*req.Gateway); err == nil {
				ipamCfg.Gateway = a
			}
		}
		opts.IPAM = &network.IPAM{Driver: "default", Config: []network.IPAMConfig{ipamCfg}}
	}
	return docker.CreateNetwork(st.Docker, ctx, req.Name, opts)
}

// NetworkRemove 删除网络
func NetworkRemove(st *state.AppState, ctx context.Context, id string) error {
	return docker.RemoveNetwork(st.Docker, ctx, id, client.NetworkRemoveOptions{})
}

// NetworksPrune 清理未使用网络
func NetworksPrune(st *state.AppState, ctx context.Context) (network.PruneReport, error) {
	return docker.PruneNetworks(st.Docker, ctx, client.NetworkPruneOptions{})
}

// Ptr 泛型指针(等价旧 api 包 ptr)
func Ptr[T any](v T) *T { return &v }
