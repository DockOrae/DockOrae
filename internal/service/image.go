package service

import (
	"context"
	"strings"

	"github.com/moby/moby/api/types/image"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/docker"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ImagesList 镜像列表(精简字段)
func ImagesList(st *state.AppState, ctx context.Context) ([]model.ImageListItem, error) {
	items, err := docker.ListImages(st.Docker, ctx, client.ImageListOptions{})
	if err != nil {
		return nil, err
	}
	return model.ToImageItems(items), nil
}

// ImageInspect 镜像详情
func ImageInspect(st *state.AppState, ctx context.Context, id string) (image.InspectResponse, error) {
	return docker.InspectImage(st.Docker, ctx, id)
}

// ImagePullRef 组装镜像引用(镜像名已带 tag 时不拼接)
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

// ImagePull 拉取镜像,返回进度流(调用方消费 NDJSON)
func ImagePull(st *state.AppState, ctx context.Context, ref string) (client.ImagePullResponse, error) {
	return docker.PullImage(st.Docker, ctx, ref, client.ImagePullOptions{})
}

// ImageRemove 删除镜像
func ImageRemove(st *state.AppState, ctx context.Context, id string, force bool) error {
	return docker.RemoveImage(st.Docker, ctx, id, client.ImageRemoveOptions{
		Force:         force,
		PruneChildren: false,
	})
}

// ImagesPrune 清理未使用镜像(dangling=false 保留 tag 镜像)
func ImagesPrune(st *state.AppState, ctx context.Context) (image.PruneReport, error) {
	return docker.PruneImages(st.Docker, ctx, client.ImagePruneOptions{})
}

// ImageTag 打标签
func ImageTag(st *state.AppState, ctx context.Context, id, repo, tag string) error {
	return docker.TagImage(st.Docker, ctx, id, repo+":"+tag)
}
