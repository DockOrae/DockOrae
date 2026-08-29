package service

import (
	"context"

	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"

	"github.com/DockerManger/Docker_Manager_Go/internal/docker"
	"github.com/DockerManger/Docker_Manager_Go/internal/model"
	"github.com/DockerManger/Docker_Manager_Go/internal/state"
)

// VolumeService 卷业务:依赖注入(docker client)
type VolumeService struct {
	docker *client.Client
}

// NewVolumeService 生产构造:从 AppState 提取实际依赖
func NewVolumeService(st *state.AppState) *VolumeService {
	return &VolumeService{docker: st.Docker}
}

// List 卷列表
func (s *VolumeService) List(ctx context.Context) ([]volume.Volume, error) {
	return docker.ListVolumes(s.docker, ctx, client.VolumeListOptions{})
}

// Raw 卷详情原始 JSON
func (s *VolumeService) Raw(ctx context.Context, name string) ([]byte, error) {
	_, raw, err := docker.InspectVolume(s.docker, ctx, name)
	return raw, err
}

// Create 创建卷(固定 local 驱动;NFS 卷由前端生成 local+driver_opts)
func (s *VolumeService) Create(ctx context.Context, req model.CreateVolumeReq) (volume.Volume, error) {
	if req.Name == "" {
		return volume.Volume{}, BadRequest("volume.nameEmpty")
	}
	return docker.CreateVolume(s.docker, ctx, client.VolumeCreateOptions{
		Name:       req.Name,
		Driver:     "local",
		DriverOpts: req.DriverOpts,
		Labels:     req.Labels,
	})
}

// Remove 删除卷
func (s *VolumeService) Remove(ctx context.Context, name string, force bool) error {
	return docker.RemoveVolume(s.docker, ctx, name, force)
}

// Prune 清理未使用卷
func (s *VolumeService) Prune(ctx context.Context) (volume.PruneReport, error) {
	return docker.PruneVolumes(s.docker, ctx, client.VolumePruneOptions{})
}
