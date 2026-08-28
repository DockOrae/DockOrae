// Package docker 分层架构中的 Docker 访问层:集中封装 moby SDK 调用,
// 上层(service)不直接接触 moby client,便于未来替换 SDK。
package docker

import (
	"os"
	"runtime"

	"github.com/moby/moby/client"
)

// NewClient 构造 moby client:DOCKER_HOST 优先,Windows 开发默认 TCP,Linux 默认 unix socket。
// 惰性连接(构造不访问 daemon),无 Docker 环境也能启动面板。
func NewClient() (*client.Client, error) {
	if host := os.Getenv("DOCKER_HOST"); host != "" {
		return client.NewClientWithOpts(client.WithHost(host))
	}
	if runtime.GOOS == "windows" {
		// Windows 本地开发默认 TCP,无 Docker 也能启动面板
		return client.NewClientWithOpts(client.WithHost("tcp://127.0.0.1:2375"))
	}
	return client.NewClientWithOpts()
}
