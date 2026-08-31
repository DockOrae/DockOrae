package service

import (
	"context"

	"github.com/DockOrae/DockOrae/internal/agent"
	"github.com/DockOrae/DockOrae/internal/model"
	"github.com/DockOrae/DockOrae/internal/state"
)

// VolumeService 卷业务(§10:执行全部在 Agent)。
type VolumeService struct {
	agent *agent.Client
}

// NewVolumeService 生产构造:从 AppState 提取实际依赖
func NewVolumeService(st *state.AppState) *VolumeService {
	return &VolumeService{agent: st.Agent}
}

// List 卷列表
func (s *VolumeService) List(ctx context.Context) ([]agent.VolumeInfo, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	return s.agent.VolumeList(ctx)
}

// Raw 卷详情原始 JSON
func (s *VolumeService) Raw(ctx context.Context, name string) ([]byte, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	return s.agent.VolumeInspectRaw(ctx, name)
}

// Create 创建卷(local 驱动;NFS 卷由前端生成 driver_opts)
func (s *VolumeService) Create(ctx context.Context, req model.CreateVolumeReq) (agent.VolumeInfo, error) {
	if req.Name == "" {
		return agent.VolumeInfo{}, BadRequest("volume.nameEmpty")
	}
	if s.agent == nil {
		return agent.VolumeInfo{}, agentUnavailable()
	}
	return s.agent.VolumeCreate(ctx, req.Name, "local", req.DriverOpts, req.Labels)
}

// Remove 删除卷
func (s *VolumeService) Remove(ctx context.Context, name string, force bool) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.VolumeRemove(ctx, name, force)
}

// Prune 清理未使用卷
func (s *VolumeService) Prune(ctx context.Context) (agent.PruneReport, error) {
	if s.agent == nil {
		return agent.PruneReport{}, agentUnavailable()
	}
	return s.agent.VolumePrune(ctx)
}
