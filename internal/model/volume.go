package model

// ---- 创建卷请求 ----

type CreateVolumeReq struct {
	Name       string            `json:"name"`
	Driver     *string           `json:"driver"`
	DriverOpts map[string]string `json:"driver_opts"`
	Labels     map[string]string `json:"labels"`
}
