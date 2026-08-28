package service

import (
	"context"

	"github.com/moby/moby/api/types/volume"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/docker"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// VolumesList 卷列表
func VolumesList(st *state.AppState, ctx context.Context) ([]volume.Volume, error) {
	return docker.ListVolumes(st.Docker, ctx, client.VolumeListOptions{})
}

// VolumeRaw 卷详情原始 JSON
func VolumeRaw(st *state.AppState, ctx context.Context, name string) ([]byte, error) {
	_, raw, err := docker.InspectVolume(st.Docker, ctx, name)
	return raw, err
}

// VolumeCreate 创建卷(支持驱动选项 driver_opts 与标签;NFS 卷由前端生成 local+driver_opts)
func VolumeCreate(st *state.AppState, ctx context.Context, req model.CreateVolumeReq) (volume.Volume, error) {
	if req.Name == "" {
		return volume.Volume{}, BadRequest("volume.nameEmpty")
	}
	driver := "local"
	if req.Driver != nil && *req.Driver != "" {
		driver = *req.Driver
	}
	return docker.CreateVolume(st.Docker, ctx, client.VolumeCreateOptions{
		Name:       req.Name,
		Driver:     driver,
		DriverOpts: req.DriverOpts,
		Labels:     req.Labels,
	})
}

// VolumeDrivers 可用卷驱动列表:local + 已启用的插件驱动(供前端下拉)
func VolumeDrivers(st *state.AppState, ctx context.Context) ([]string, error) {
	drivers := []string{"local"}
	plugins, err := docker.ListPlugins(st.Docker, ctx)
	if err == nil {
		for _, p := range plugins {
			if p.Enabled {
				drivers = append(drivers, p.Name)
			}
		}
	}
	return drivers, nil
}

// VolumeRemove 删除卷
func VolumeRemove(st *state.AppState, ctx context.Context, name string, force bool) error {
	return docker.RemoveVolume(st.Docker, ctx, name, force)
}

// VolumesPrune 清理未使用卷
func VolumesPrune(st *state.AppState, ctx context.Context) (volume.PruneReport, error) {
	return docker.PruneVolumes(st.Docker, ctx, client.VolumePruneOptions{})
}
