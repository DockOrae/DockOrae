package service

import (
	"context"

	"github.com/DockOrae/DockOrae/internal/agent"
	"github.com/DockOrae/DockOrae/internal/model"
	"github.com/DockOrae/DockOrae/internal/state"
)

// NetworkService 网络业务(§9:执行全部在 Agent)。
type NetworkService struct {
	agent *agent.Client
}

// NewNetworkService 生产构造:从 AppState 提取实际依赖
func NewNetworkService(st *state.AppState) *NetworkService {
	return &NetworkService{agent: st.Agent}
}

// List 网络列表(精简字段)
func (s *NetworkService) List(ctx context.Context) ([]model.NetworkListItem, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	items, err := s.agent.NetworkList(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.NetworkListItem, 0, len(items))
	for _, it := range items {
		out = append(out, model.NetworkListItem{
			ID: it.ID, Name: it.Name, Driver: it.Driver, Scope: it.Scope, IPAM: it.IPAM,
		})
	}
	return out, nil
}

// Raw 网络详情原始 JSON(handler 透传)
func (s *NetworkService) Raw(ctx context.Context, id string) ([]byte, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	return s.agent.NetworkInspectRaw(ctx, id)
}

// Create 创建网络
func (s *NetworkService) Create(ctx context.Context, req model.CreateNetworkReq) (string, error) {
	if req.Name == "" {
		return "", BadRequest("network.nameEmpty")
	}
	if s.agent == nil {
		return "", agentUnavailable()
	}
	driver := "bridge"
	if req.Driver != nil && *req.Driver != "" {
		driver = *req.Driver
	}
	return s.agent.NetworkCreate(ctx, agent.NetworkCreateReq{
		Name:     req.Name,
		Driver:   driver,
		Internal: req.Internal != nil && *req.Internal,
		Subnet:   req.Subnet,
		Gateway:  req.Gateway,
	})
}

// Remove 删除网络
func (s *NetworkService) Remove(ctx context.Context, id string) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.NetworkRemove(ctx, id)
}

// Prune 清理未使用网络
func (s *NetworkService) Prune(ctx context.Context) (agent.PruneReport, error) {
	if s.agent == nil {
		return agent.PruneReport{}, agentUnavailable()
	}
	return s.agent.NetworkPrune(ctx)
}
