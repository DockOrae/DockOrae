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

// NetworkService 网络业务:依赖注入(docker client)
type NetworkService struct {
	docker *client.Client
}

// NewNetworkService 生产构造:从 AppState 提取实际依赖
func NewNetworkService(st *state.AppState) *NetworkService {
	return &NetworkService{docker: st.Docker}
}

// List 网络列表(精简字段)
func (s *NetworkService) List(ctx context.Context) ([]model.NetworkListItem, error) {
	items, err := docker.ListNetworks(s.docker, ctx, client.NetworkListOptions{})
	if err != nil {
		return nil, err
	}
	return model.ToNetworkItems(items), nil
}

// Inspect 网络详情(原始 JSON 透传)
func (s *NetworkService) Inspect(ctx context.Context, id string) (network.Inspect, error) {
	net, _, err := docker.InspectNetwork(s.docker, ctx, id)
	return net, err
}

// Raw 网络详情原始 JSON(handler 透传)
func (s *NetworkService) Raw(ctx context.Context, id string) ([]byte, error) {
	_, raw, err := docker.InspectNetwork(s.docker, ctx, id)
	return raw, err
}

// Create 创建网络
func (s *NetworkService) Create(ctx context.Context, req model.CreateNetworkReq) (string, error) {
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
	return docker.CreateNetwork(s.docker, ctx, req.Name, opts)
}

// Remove 删除网络
func (s *NetworkService) Remove(ctx context.Context, id string) error {
	return docker.RemoveNetwork(s.docker, ctx, id, client.NetworkRemoveOptions{})
}

// Prune 清理未使用网络
func (s *NetworkService) Prune(ctx context.Context) (network.PruneReport, error) {
	return docker.PruneNetworks(s.docker, ctx, client.NetworkPruneOptions{})
}

// Ptr 泛型指针(等价旧 api 包 ptr)
func Ptr[T any](v T) *T { return &v }
