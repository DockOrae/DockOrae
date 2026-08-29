package model

// ---- 在线更新 ----

// ReleaseAsset GitHub Release 资产
type ReleaseAsset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"browser_download_url"`
}

type UpdateRelease struct {
	TagName     string         `json:"tag_name"`
	Name        string         `json:"name"`
	Body        string         `json:"body"`
	PublishedAt string         `json:"published_at"`
	HTMLURL     string         `json:"html_url"`
	Prerelease  bool           `json:"prerelease"`
	Draft       bool           `json:"draft"`
	Assets      []ReleaseAsset `json:"assets"`
}

// ReleaseNoteSection Release Notes 分类段(解析失败时为空,前端回退显示原始 body)
type ReleaseNoteSection struct {
	Type  string   `json:"type"` // features / bug_fixes / improvements / security / breaking_changes
	Items []string `json:"items"`
}

// UpdateInfo 更新检查结果(前端弹框展示 + 可用性判断)
type UpdateInfo struct {
	Current              string               `json:"current"`                 // 展示用(v1.0.3 / unknown)
	Latest               string               `json:"latest"`                  // 展示用(v1.0.3)
	HasUpdate            bool                 `json:"has_update"`
	Release              *UpdateRelease       `json:"release,omitempty"`
	InstallType          string               `json:"install_type"`                       // binary / docker
	Installable          bool                 `json:"installable"`                        // 当前安装方式是否有可用更新包
	NotInstallableReason string               `json:"not_installable_reason,omitempty"`   // 不可用原因(镜像未发布/无对应 asset)
	Notes                []ReleaseNoteSection `json:"notes,omitempty"`                    // 分类解析结果
	NotesRaw             bool                 `json:"notes_raw"`                          // true=分类解析失败,前端显示 release.body 原文
	Error                string               `json:"error,omitempty"`
}
