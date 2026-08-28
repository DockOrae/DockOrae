// Package model 分层架构中的数据模型层:请求/响应结构体与类型映射。
// 仅依赖 moby 类型,不含任何业务逻辑。
package model

import (
	"github.com/moby/moby/api/types/container"
)

// ---- 容器列表(精简) ----
// moby 全量 Summary 携带 NetworkSettings/Mounts 等大字段,直传跨洋响应可达数百 KB;
// 列表接口只返回前端实际使用的字段,响应体积缩小 5~10 倍。

type ContainerMountItem struct {
	Type string `json:"Type"`
	Name string `json:"Name"`
}

type ContainerListItem struct {
	ID      string                  `json:"Id"`
	Names   []string                `json:"Names"`
	Image   string                  `json:"Image"`
	State   string                  `json:"State"`
	Ports   []container.PortSummary `json:"Ports"`
	Created int64                   `json:"Created"`
	Mounts  []ContainerMountItem    `json:"Mounts"`
	Labels  map[string]string       `json:"Labels,omitempty"`
}

func ToContainerItems(items []container.Summary) []ContainerListItem {
	out := make([]ContainerListItem, 0, len(items))
	for _, it := range items {
		mounts := make([]ContainerMountItem, 0, len(it.Mounts))
		for _, m := range it.Mounts {
			mounts = append(mounts, ContainerMountItem{Type: string(m.Type), Name: m.Name})
		}
		out = append(out, ContainerListItem{
			ID:      it.ID,
			Names:   it.Names,
			Image:   it.Image,
			State:   string(it.State),
			Ports:   it.Ports,
			Created: it.Created,
			Mounts:  mounts,
			Labels:  it.Labels,
		})
	}
	return out
}

// ---- 创建容器请求 ----

type PortMap struct {
	Container string  `json:"container"`
	Host      uint16  `json:"host"`
	HostIP    *string `json:"host_ip"`
}

type VolumeMap struct {
	Host      *string `json:"host"`
	Volume    *string `json:"volume"`
	Container string  `json:"container"`
	Mode      *string `json:"mode"`
}

type CreateContainerReq struct {
	Name          string      `json:"name"`
	Image         string      `json:"image"`
	Cmd           []string    `json:"cmd"`
	Env           []string    `json:"env"`
	Ports         []PortMap   `json:"ports"`
	Volumes       []VolumeMap `json:"volumes"`
	Network       *string     `json:"network"`
	RestartPolicy *string     `json:"restart_policy"`
	Tty           *bool       `json:"tty"`
	Privileged    *bool       `json:"privileged"`
}
