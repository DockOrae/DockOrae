// Agent Client Docker 资源操作(§5:所有 Docker 能力统一由 Agent 实现)。
// 本文件为 internal/agent 包内 typed 方法:固定端点 + 类型化请求/响应。
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ---------- Containers ----------

// ContainerList 容器列表(all=true 含已停止)
func (c *Client) ContainerList(ctx context.Context, all bool) ([]ContainerSummary, error) {
	path := "/v1/docker/containers"
	if all {
		path += "?all=1"
	}
	data, err := c.Call(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return nil, err
	}
	var out []ContainerSummary
	if err := decodeItems(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ContainerInspectRaw 容器详情原始 JSON(透传前端 + 重建容器)
func (c *Client) ContainerInspectRaw(ctx context.Context, id string) (json.RawMessage, error) {
	data, err := c.Call(ctx, http.MethodGet, "/v1/docker/containers/"+id, nil, "")
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

// ContainerCreateReq 创建容器请求(config/host_config/networking_config 为 docker API 原始 JSON)
type ContainerCreateReq struct {
	Name             string          `json:"name"`
	Config           json.RawMessage `json:"config"`
	HostConfig       json.RawMessage `json:"host_config"`
	NetworkingConfig json.RawMessage `json:"networking_config"`
}

// ContainerCreate 创建容器,返回容器 ID
func (c *Client) ContainerCreate(ctx context.Context, req ContainerCreateReq) (string, error) {
	data, err := c.Call(ctx, http.MethodPost, "/v1/docker/containers", req, "")
	if err != nil {
		return "", err
	}
	id, _ := data["id"].(string)
	return id, nil
}

func (c *Client) ContainerStart(ctx context.Context, id string) error {
	return c.simplePost(ctx, "/v1/docker/containers/"+id+"/start", nil)
}
func (c *Client) ContainerStop(ctx context.Context, id string, timeout *int) error {
	path := "/v1/docker/containers/" + id + "/stop"
	if timeout != nil {
		path += fmt.Sprintf("?timeout=%d", *timeout)
	}
	return c.simplePost(ctx, path, nil)
}
func (c *Client) ContainerRestart(ctx context.Context, id string, timeout *int) error {
	path := "/v1/docker/containers/" + id + "/restart"
	if timeout != nil {
		path += fmt.Sprintf("?timeout=%d", *timeout)
	}
	return c.simplePost(ctx, path, nil)
}
func (c *Client) ContainerKill(ctx context.Context, id string) error {
	return c.simplePost(ctx, "/v1/docker/containers/"+id+"/kill", nil)
}
func (c *Client) ContainerPause(ctx context.Context, id string) error {
	return c.simplePost(ctx, "/v1/docker/containers/"+id+"/pause", nil)
}
func (c *Client) ContainerUnpause(ctx context.Context, id string) error {
	return c.simplePost(ctx, "/v1/docker/containers/"+id+"/unpause", nil)
}
func (c *Client) ContainerRename(ctx context.Context, id, name string) error {
	return c.simplePost(ctx, "/v1/docker/containers/"+id+"/rename", map[string]any{"name": name})
}

// ContainerRemove 删除容器
func (c *Client) ContainerRemove(ctx context.Context, id string, force, removeVolumes bool) error {
	path := "/v1/docker/containers/" + id
	sep := "?"
	if force {
		path += sep + "force=1"
		sep = "&"
	}
	if removeVolumes {
		path += sep + "v=1"
	}
	return c.simpleDelete(ctx, path)
}

// ContainerPrune 清理已停止容器(需 confirm)
func (c *Client) ContainerPrune(ctx context.Context) (PruneReport, error) {
	var rep PruneReport
	data, err := c.Call(ctx, http.MethodPost, "/v1/docker/containers/prune", map[string]any{"confirm": true}, "")
	if err != nil {
		return rep, err
	}
	rep.ContainersDeleted = strSlice(data["containers_deleted"])
	rep.SpaceReclaimed = int64Of(data["space_reclaimed"])
	return rep, nil
}

// ContainerWait 等待容器退出(在线更新 helper 用)
func (c *Client) ContainerWait(ctx context.Context, id string) (int64, error) {
	data, err := c.Call(ctx, http.MethodGet, "/v1/docker/containers/"+id+"/wait", nil, "")
	if err != nil {
		return 0, err
	}
	return int64Of(data["status_code"]), nil
}

// ---------- Images ----------

// ImageList 镜像列表
func (c *Client) ImageList(ctx context.Context) ([]ImageSummary, error) {
	data, err := c.Call(ctx, http.MethodGet, "/v1/docker/images", nil, "")
	if err != nil {
		return nil, err
	}
	var out []ImageSummary
	if err := decodeItems(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// ImageInspectRaw 镜像详情原始 JSON
func (c *Client) ImageInspectRaw(ctx context.Context, id string) (json.RawMessage, error) {
	data, err := c.Call(ctx, http.MethodGet, "/v1/docker/images/"+id, nil, "")
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

// ImageRemove 删除镜像
func (c *Client) ImageRemove(ctx context.Context, id string, force bool) error {
	path := "/v1/docker/images/" + id
	if force {
		path += "?force=1"
	}
	return c.simpleDelete(ctx, path)
}

// ImagePrune 清理悬空镜像
func (c *Client) ImagePrune(ctx context.Context) (PruneReport, error) {
	var rep PruneReport
	data, err := c.Call(ctx, http.MethodPost, "/v1/docker/images/prune", map[string]any{"confirm": true}, "")
	if err != nil {
		return rep, err
	}
	rep.ImagesDeleted = strSlice(data["images_deleted"])
	rep.SpaceReclaimed = int64Of(data["space_reclaimed"])
	return rep, nil
}

// ImageTag 打标签
func (c *Client) ImageTag(ctx context.Context, id, repo, tag string) error {
	return c.simplePost(ctx, "/v1/docker/images/"+id+"/tag", map[string]any{"repo": repo, "tag": tag})
}

// ---------- Networks ----------

// NetworkList 网络列表
func (c *Client) NetworkList(ctx context.Context) ([]NetworkSummary, error) {
	data, err := c.Call(ctx, http.MethodGet, "/v1/docker/networks", nil, "")
	if err != nil {
		return nil, err
	}
	var out []NetworkSummary
	if err := decodeItems(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// NetworkInspectRaw 网络详情原始 JSON
func (c *Client) NetworkInspectRaw(ctx context.Context, id string) (json.RawMessage, error) {
	data, err := c.Call(ctx, http.MethodGet, "/v1/docker/networks/"+id, nil, "")
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

// NetworkCreateReq 创建网络请求
type NetworkCreateReq struct {
	Name     string  `json:"name"`
	Driver   string  `json:"driver"`
	Internal bool    `json:"internal"`
	Subnet   *string `json:"subnet,omitempty"`
	Gateway  *string `json:"gateway,omitempty"`
}

// NetworkCreate 创建网络
func (c *Client) NetworkCreate(ctx context.Context, req NetworkCreateReq) (string, error) {
	data, err := c.Call(ctx, http.MethodPost, "/v1/docker/networks", req, "")
	if err != nil {
		return "", err
	}
	id, _ := data["id"].(string)
	return id, nil
}

// NetworkRemove 删除网络
func (c *Client) NetworkRemove(ctx context.Context, id string) error {
	return c.simpleDelete(ctx, "/v1/docker/networks/"+id)
}

// NetworkPrune 清理未使用网络
func (c *Client) NetworkPrune(ctx context.Context) (PruneReport, error) {
	var rep PruneReport
	data, err := c.Call(ctx, http.MethodPost, "/v1/docker/networks/prune", map[string]any{"confirm": true}, "")
	if err != nil {
		return rep, err
	}
	rep.NetworksDeleted = strSlice(data["networks_deleted"])
	return rep, nil
}

// NetworkConnect 容器接入网络
func (c *Client) NetworkConnect(ctx context.Context, id, container string) error {
	return c.simplePost(ctx, "/v1/docker/networks/"+id+"/connect", map[string]any{"container": container})
}

// NetworkDisconnect 容器断开网络
func (c *Client) NetworkDisconnect(ctx context.Context, id, container string, force bool) error {
	return c.simplePost(ctx, "/v1/docker/networks/"+id+"/disconnect", map[string]any{"container": container, "force": force})
}

// ---------- Volumes ----------

// VolumeList 卷列表
func (c *Client) VolumeList(ctx context.Context) ([]VolumeInfo, error) {
	data, err := c.Call(ctx, http.MethodGet, "/v1/docker/volumes", nil, "")
	if err != nil {
		return nil, err
	}
	var out []VolumeInfo
	if err := decodeItems(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// VolumeInspectRaw 卷详情原始 JSON
func (c *Client) VolumeInspectRaw(ctx context.Context, name string) (json.RawMessage, error) {
	data, err := c.Call(ctx, http.MethodGet, "/v1/docker/volumes/"+name, nil, "")
	if err != nil {
		return nil, err
	}
	return json.Marshal(data)
}

// VolumeCreate 创建卷
func (c *Client) VolumeCreate(ctx context.Context, name string, driver string, driverOpts, labels map[string]string) (VolumeInfo, error) {
	payload := map[string]any{"name": name, "driver": driver, "driver_opts": driverOpts, "labels": labels}
	data, err := c.Call(ctx, http.MethodPost, "/v1/docker/volumes", payload, "")
	if err != nil {
		return VolumeInfo{}, err
	}
	raw, _ := json.Marshal(data)
	var vol VolumeInfo
	_ = json.Unmarshal(raw, &vol)
	return vol, nil
}

// VolumeRemove 删除卷
func (c *Client) VolumeRemove(ctx context.Context, name string, force bool) error {
	path := "/v1/docker/volumes/" + name
	if force {
		path += "?force=1"
	}
	return c.simpleDelete(ctx, path)
}

// VolumePrune 清理未使用卷
func (c *Client) VolumePrune(ctx context.Context) (PruneReport, error) {
	var rep PruneReport
	data, err := c.Call(ctx, http.MethodPost, "/v1/docker/volumes/prune", map[string]any{"confirm": true}, "")
	if err != nil {
		return rep, err
	}
	rep.VolumesDeleted = strSlice(data["volumes_deleted"])
	rep.SpaceReclaimed = int64Of(data["space_reclaimed"])
	return rep, nil
}

// ---------- 通用 ----------

// simplePost 无返回体 POST
func (c *Client) simplePost(ctx context.Context, path string, payload any) error {
	_, err := c.Call(ctx, http.MethodPost, path, payload, "")
	return err
}

// simpleDelete DELETE
func (c *Client) simpleDelete(ctx context.Context, path string) error {
	_, err := c.Call(ctx, http.MethodDelete, path, nil, "")
	return err
}

// decodeItems 解码 {"items":[...]}
func decodeItems(data map[string]any, out any) error {
	raw, err := json.Marshal(data["items"])
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func strSlice(v any) []string {
	raw, _ := json.Marshal(v)
	var out []string
	_ = json.Unmarshal(raw, &out)
	return out
}

func int64Of(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	}
	return 0
}
