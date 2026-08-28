package model

import (
	"github.com/moby/moby/api/types/image"
)

// ---- 镜像列表(精简) ----

type ImageListItem struct {
	ID       string   `json:"Id"`
	RepoTags []string `json:"RepoTags"`
	Size     int64    `json:"Size"`
	Created  int64    `json:"Created"`
}

func ToImageItems(items []image.Summary) []ImageListItem {
	out := make([]ImageListItem, 0, len(items))
	for _, it := range items {
		out = append(out, ImageListItem{ID: it.ID, RepoTags: it.RepoTags, Size: it.Size, Created: it.Created})
	}
	return out
}

// ---- 拉取镜像请求 ----

type PullImageReq struct {
	FromImage string  `json:"from_image"`
	Tag       *string `json:"tag"`
}
