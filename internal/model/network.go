package model

import (
	"github.com/moby/moby/api/types/network"
)

// ---- 网络列表(精简) ----
// 新版 moby /networks 不返回容器端点信息;仅返回列表展示字段

type NetworkListItem struct {
	ID     string       `json:"Id"`
	Name   string       `json:"Name"`
	Driver string       `json:"Driver"`
	Scope  string       `json:"Scope"`
	IPAM   network.IPAM `json:"IPAM"`
}

func ToNetworkItems(items []network.Summary) []NetworkListItem {
	out := make([]NetworkListItem, 0, len(items))
	for _, it := range items {
		out = append(out, NetworkListItem{
			ID: it.ID, Name: it.Name, Driver: it.Driver, Scope: it.Scope, IPAM: it.IPAM,
		})
	}
	return out
}

// ---- 创建网络请求 ----

type CreateNetworkReq struct {
	Name     string  `json:"name"`
	Driver   *string `json:"driver"`
	Subnet   *string `json:"subnet"`
	Gateway  *string `json:"gateway"`
	Internal *bool   `json:"internal"`
}
