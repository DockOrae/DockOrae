package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/DockOrae/DockOrae/internal/agent"
	"github.com/DockOrae/DockOrae/internal/model"
	"github.com/DockOrae/DockOrae/internal/state"
)

// ImageService 镜像业务(§8:执行全部在 Agent)。
type ImageService struct {
	agent *agent.Client
}

// NewImageService 生产构造:从 AppState 提取实际依赖
func NewImageService(st *state.AppState) *ImageService {
	return &ImageService{agent: st.Agent}
}

// List 镜像列表(精简字段)
func (s *ImageService) List(ctx context.Context) ([]model.ImageListItem, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	items, err := s.agent.ImageList(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]model.ImageListItem, 0, len(items))
	for _, it := range items {
		out = append(out, model.ImageListItem{
			ID: it.ID, RepoTags: it.RepoTags, Size: it.Size, Created: it.Created,
		})
	}
	return out, nil
}

// Inspect 镜像详情(原始 JSON)
func (s *ImageService) Inspect(ctx context.Context, id string) (json.RawMessage, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	return s.agent.ImageInspectRaw(ctx, id)
}

// PullStream 拉取镜像,返回 NDJSON 进度流(调用方转发前端)
func (s *ImageService) PullStream(ctx context.Context, ref string) (*agent.StreamBody, error) {
	if s.agent == nil {
		return nil, agentUnavailable()
	}
	return s.agent.PullImageStream(ctx, ref)
}

// Remove 删除镜像
func (s *ImageService) Remove(ctx context.Context, id string, force bool) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.ImageRemove(ctx, id, force)
}

// Prune 清理未使用镜像
func (s *ImageService) Prune(ctx context.Context) (agent.PruneReport, error) {
	if s.agent == nil {
		return agent.PruneReport{}, agentUnavailable()
	}
	return s.agent.ImagePrune(ctx)
}

// Tag 打标签
func (s *ImageService) Tag(ctx context.Context, id, repo, tag string) error {
	if s.agent == nil {
		return agentUnavailable()
	}
	return s.agent.ImageTag(ctx, id, repo, tag)
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
