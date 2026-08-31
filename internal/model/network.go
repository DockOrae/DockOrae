package model

import "encoding/json"

// ---- 网络列表(精简) ----

type NetworkListItem struct {
	ID     string          `json:"Id"`
	Name   string          `json:"Name"`
	Driver string          `json:"Driver"`
	Scope  string          `json:"Scope"`
	IPAM   json.RawMessage `json:"IPAM,omitempty"`
}

func ToNetworkItems(items []NetworkListItem) []NetworkListItem {
	return items
}

// ---- 创建网络请求 ----

type CreateNetworkReq struct {
	Name     string  `json:"name"`
	Driver   *string `json:"driver"`
	Subnet   *string `json:"subnet"`
	Gateway  *string `json:"gateway"`
	Internal *bool   `json:"internal"`
}
