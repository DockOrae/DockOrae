package model

// ---- 在线更新 ----

type UpdateRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Prerelease  bool   `json:"prerelease"`
}

type UpdateInfo struct {
	Current   string         `json:"current"`
	Latest    string         `json:"latest"`
	HasUpdate bool           `json:"has_update"`
	Release   *UpdateRelease `json:"release,omitempty"`
	Error     string         `json:"error,omitempty"`
}
