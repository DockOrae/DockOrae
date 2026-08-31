// Package agent DockOrae 主程序侧的 Agent 客户端类型定义。
// 数据契约:Agent 返回 docker API 原始 JSON 结构(DockOrae 不再依赖 moby SDK),
// 本文件定义解码所需的精简 DTO。
package agent

import (
	"encoding/json"
)

// ContainerPort 容器端口(映射自 docker container.Summary.Ports)
type ContainerPort struct {
	IP          string `json:"IP"`
	PrivatePort uint16 `json:"PrivatePort"`
	PublicPort  uint16 `json:"PublicPort"`
	Type        string `json:"Type"`
}

// ContainerMount 容器挂载(精简)
type ContainerMount struct {
	Type string `json:"Type"`
	Name string `json:"Name"`
}

// ContainerSummary 容器列表项(docker container.Summary 兼容 JSON)
type ContainerSummary struct {
	ID      string            `json:"Id"`
	Names   []string          `json:"Names"`
	Image   string            `json:"Image"`
	State   string            `json:"State"`
	Status  string            `json:"Status"`
	Ports   []ContainerPort   `json:"Ports"`
	Created int64             `json:"Created"`
	Mounts  []ContainerMount  `json:"Mounts"`
	Labels  map[string]string `json:"Labels"`
}

// ImageSummary 镜像列表项(docker image.Summary 兼容 JSON)
type ImageSummary struct {
	ID          string            `json:"Id"`
	RepoTags    []string          `json:"RepoTags"`
	RepoDigests []string          `json:"RepoDigests"`
	Created     int64             `json:"Created"`
	Size        int64             `json:"Size"`
	Labels      map[string]string `json:"Labels"`
	Containers  int64             `json:"Containers"`
}

// NetworkSummary 网络列表项(docker network.Summary 兼容 JSON)
type NetworkSummary struct {
	Name       string            `json:"Name"`
	ID         string            `json:"Id"`
	Driver     string            `json:"Driver"`
	Scope      string            `json:"Scope"`
	Internal   bool              `json:"Internal"`
	Attachable bool              `json:"Attachable"`
	Labels     map[string]string `json:"Labels"`
	Created    string            `json:"Created"`
	IPAM       json.RawMessage   `json:"IPAM,omitempty"`
}

// VolumeInfo 卷信息(docker volume.Volume 兼容 JSON)
type VolumeInfo struct {
	Name       string            `json:"Name"`
	Driver     string            `json:"Driver"`
	Mountpoint string            `json:"Mountpoint"`
	Labels     map[string]string `json:"Labels"`
	Scope      string            `json:"Scope"`
	Options    map[string]string `json:"Options"`
}

// PruneReport 清理报告
type PruneReport struct {
	Deleted           []string `json:"Deleted,omitempty"`
	SpaceReclaimed    int64    `json:"SpaceReclaimed"`
	ContainersDeleted []string `json:"ContainersDeleted,omitempty"`
	ImagesDeleted     []string `json:"ImagesDeleted,omitempty"`
	NetworksDeleted   []string `json:"NetworksDeleted,omitempty"`
	VolumesDeleted    []string `json:"VolumesDeleted,omitempty"`
}

// ComposeRunResult compose 同步执行结果
type ComposeRunResult struct {
	OK     bool   `json:"ok"`
	Output string `json:"output"`
}

// ComposeProjectFile compose 项目附加文件(相对路径 → base64)
type ComposeProjectFile struct {
	Files map[string]string `json:"files"`
}

// StatsFrame 容器 stats 原始帧(仅解码面板计算所需字段)
type StatsFrame struct {
	CPUStats struct {
		CPUUsage struct {
			TotalUsage uint64 `json:"total_usage"`
		} `json:"cpu_usage"`
		SystemUsage uint64 `json:"system_cpu_usage"`
		OnlineCPUs  uint32 `json:"online_cpus"`
	} `json:"cpu_stats"`
	MemoryStats struct {
		Usage uint64 `json:"usage"`
		Limit uint64 `json:"limit"`
	} `json:"memory_stats"`
	Networks map[string]struct {
		RxBytes uint64 `json:"rx_bytes"`
		TxBytes uint64 `json:"tx_bytes"`
	} `json:"networks"`
	PidsStats struct {
		Current uint64 `json:"current"`
	} `json:"pids_stats"`
}

// EventMessage docker 事件(与 docker events API 消息 JSON 兼容)
type EventMessage struct {
	Type   string `json:"Type"`
	Action string `json:"Action"`
	Actor  struct {
		ID         string            `json:"ID"`
		Attributes map[string]string `json:"Attributes"`
	} `json:"Actor"`
	Scope    string `json:"scope,omitempty"`
	Time     int64  `json:"time,omitempty"`
	TimeNano int64  `json:"timeNano,omitempty"`
}
