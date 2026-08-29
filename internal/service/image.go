package service

import (
	"context"
	"strings"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"

	"github.com/DockerManger/Docker_Manager_Go/internal/docker"
	"github.com/DockerManger/Docker_Manager_Go/internal/model"
	"github.com/DockerManger/Docker_Manager_Go/internal/state"
)

// ImageService 镜像业务:依赖注入(docker client)
type ImageService struct {
	docker *client.Client
}

// NewImageService 生产构造:从 AppState 提取实际依赖
func NewImageService(st *state.AppState) *ImageService {
	return &ImageService{docker: st.Docker}
}

// List 镜像列表(精简字段)
func (s *ImageService) List(ctx context.Context) ([]model.ImageListItem, error) {
	items, err := docker.ListImages(s.docker, ctx, client.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	return model.ToImageItems(items), nil
}

// Inspect 镜像详情
func (s *ImageService) Inspect(ctx context.Context, id string) (image.InspectResponse, error) {
	return docker.InspectImage(s.docker, ctx, id)
}

// Pull 拉取镜像,返回进度流(调用方消费 NDJSON)
func (s *ImageService) Pull(ctx context.Context, ref string) (client.ImagePullResponse, error) {
	return docker.PullImage(s.docker, ctx, ref, client.ImagePullOptions{})
}

// Remove 删除镜像
func (s *ImageService) Remove(ctx context.Context, id string, force bool) error {
	return docker.RemoveImage(s.docker, ctx, id, client.ImageRemoveOptions{
		Force:         force,
		PruneChildren: false,
	})
}

// Prune 清理未使用镜像(dangling=false 保留 tag 镜像)
func (s *ImageService) Prune(ctx context.Context) (image.PruneReport, error) {
	return docker.PruneImages(s.docker, ctx, client.ImagePruneOptions{})
}

// Tag 打标签
func (s *ImageService) Tag(ctx context.Context, id, repo, tag string) error {
	return docker.TagImage(s.docker, ctx, id, repo+":"+tag)
}

// ImagePullRef 组装镜像引用(镜像名已带 tag 时不拼接;纯函数)
func ImagePullRef(fromImage, tag string) string {
	ref := fromImage
	lastSlash := strings.LastIndex(ref, "/")
	namePart := ref
	if lastSlash >= 0 {
		namePart = ref[lastSlash+1:]
	}
	if !strings.Contains(namePart, ":") {
		ref = ref + ":" + tag
	}
	return ref
}
