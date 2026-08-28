package model

// ---- Compose 栈 ----

type ComposeProject struct {
	Project  string `json:"project"`
	Services int    `json:"services"`
	Running  int    `json:"running"`
	Status   string `json:"status"`
	Managed  bool   `json:"managed"`
}

type ComposeInspect struct {
	Project    string              `json:"project"`
	Managed    bool                `json:"managed"`
	Containers []ContainerListItem `json:"containers"`
	Yaml       *string             `json:"yaml"`
}
