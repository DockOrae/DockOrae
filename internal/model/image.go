package model

// ---- 镜像列表(精简) ----

type ImageListItem struct {
	ID       string   `json:"Id"`
	RepoTags []string `json:"RepoTags"`
	Size     int64    `json:"Size"`
	Created  int64    `json:"Created"`
}

// ---- 拉取镜像请求 ----

type PullImageReq struct {
	FromImage string  `json:"from_image"`
	Tag       *string `json:"tag"`
}
