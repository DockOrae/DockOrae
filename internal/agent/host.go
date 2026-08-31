// Agent Client Host/Docker 状态端点(§13-§17:宿主信息/监控/IO/镜像加速)。
package agent

import (
	"context"
	"net/http"
	"strconv"
)

// HostMonitor 宿主监控快照(cpu/mem/load/swap/disk)
func (c *Client) HostMonitor(ctx context.Context) (map[string]any, error) {
	return c.Call(ctx, http.MethodGet, "/v1/host/monitor", nil, "")
}

// DockerIO 容器网络/磁盘 IO 速率
func (c *Client) DockerIO(ctx context.Context) (map[string]any, error) {
	return c.Call(ctx, http.MethodGet, "/v1/docker/io", nil, "")
}

// RegistryMirrors 读取镜像加速配置
func (c *Client) RegistryMirrors(ctx context.Context) (map[string]any, error) {
	return c.Call(ctx, http.MethodGet, "/v1/docker/registry_mirrors", nil, "")
}

// SaveRegistryMirrors 写入镜像加速配置
func (c *Client) SaveRegistryMirrors(ctx context.Context, mirrors []string) error {
	_, err := c.Call(ctx, http.MethodPost, "/v1/docker/registry_mirrors", map[string]any{
		"mirrors": mirrors,
		"confirm": true,
	}, "")
	return err
}

// DockerVersion Docker 引擎版本
func (c *Client) DockerVersion(ctx context.Context) (map[string]any, error) {
	return c.Call(ctx, http.MethodGet, "/v1/docker/version", nil, "")
}

// DockerServiceAction Docker 服务操作(start/stop/restart)
func (c *Client) DockerServiceAction(ctx context.Context, action string) error {
	_, err := c.Call(ctx, http.MethodPost, "/v1/docker/service", map[string]any{
		"action":  action,
		"confirm": true,
	}, "")
	return err
}

// PanelCompose 读取面板宿主 compose 文件(在线更新定位 compose 目录用)
func (c *Client) PanelCompose(ctx context.Context) (dir, yaml string, err error) {
	data, err := c.Call(ctx, http.MethodGet, "/v1/panel/compose", nil, "")
	if err != nil {
		return "", "", err
	}
	dir, _ = data["dir"].(string)
	yaml, _ = data["yaml"].(string)
	return dir, yaml, nil
}

// ContainerLogsTail 容器日志尾部文本(更新失败诊断)
func (c *Client) ContainerLogsTail(ctx context.Context, id string, lines int) (string, error) {
	data, err := c.Call(ctx, http.MethodGet, "/v1/docker/containers/"+id+"/logs_tail?lines="+itoa(lines), nil, "")
	if err != nil {
		return "", err
	}
	out, _ := data["logs"].(string)
	return out, nil
}

func itoa(n int) string {
	if n <= 0 {
		return "80"
	}
	return strconv.Itoa(n)
}
