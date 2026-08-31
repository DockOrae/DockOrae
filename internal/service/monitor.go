package service

import (
	"context"
	"io"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/DockOrae/DockOrae/internal/state"
)

// ---------------- 公网 IP(仿 3x-ui:多服务列表 + 5 分钟缓存) ----------------
// 保留在面板:公网 IP 探测与 Docker/宿主无关。

var publicIPCtl = struct {
	sync.Mutex
	v4, v6 string
	at     time.Time
}{}

var ipv4Services = []string{
	"https://api4.ipify.org",
	"https://api.ipify.org",
	"https://ipv4.icanhazip.com",
}
var ipv6Services = []string{
	"https://api6.ipify.org",
	"https://ipv6.icanhazip.com",
}

// PublicIPs 返回 (ipv4, ipv6);空串 = 获取失败(前端显示 N/A)
func PublicIPs() (string, string) {
	publicIPCtl.Lock()
	defer publicIPCtl.Unlock()

	if !publicIPCtl.at.IsZero() && time.Since(publicIPCtl.at) < 5*time.Minute {
		return publicIPCtl.v4, publicIPCtl.v6
	}
	client := &http.Client{Timeout: 3 * time.Second}
	v4, v6 := "", ""
	for _, svc := range ipv4Services {
		if ip := fetchPublicIP(client, svc); ip != "" {
			v4 = ip
			break
		}
	}
	for _, svc := range ipv6Services {
		if ip := fetchPublicIP(client, svc); ip != "" {
			v6 = ip
			break
		}
	}
	publicIPCtl.v4, publicIPCtl.v6 = v4, v6
	if v4 == "" && v6 == "" {
		// 全部失败:30 秒后重试
		publicIPCtl.at = time.Now().Add(-4*time.Minute + 30*time.Second)
	} else {
		publicIPCtl.at = time.Now()
	}
	return v4, v6
}

func fetchPublicIP(client *http.Client, url string) string {
	resp, err := client.Get(url)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	// 与 3x-ui 一致:403/451 视为区域受限,不再尝试
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusUnavailableForLegalReasons {
		return ""
	}
	if resp.StatusCode != http.StatusOK {
		return ""
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 64))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

// ---------------- 宿主 / 监控快照(§13-§17:全部数据来自 Agent) ----------------

// HostInfo 宿主机信息(/system/host;Agent 执行,面板数据转换)
func HostInfo(st *state.AppState) map[string]any {
	info := map[string]any{
		"hostname":    "unknown",
		"os":          "unknown",
		"kernel":      "unknown",
		"arch":        runtime.GOARCH,
		"cpu_model":   "",
		"cpu_cores":   0,
		"mem_total":   0,
		"uptime":      0,
		"server_time": time.Now().Unix(),
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if st.Agent == nil {
		return info
	}
	if data, err := st.Agent.Call(ctx, "GET", "/v1/host/info", nil, ""); err == nil {
		for k := range info {
			if v, ok := data[k]; ok {
				info[k] = v
			}
		}
	}
	// Docker 版本(尽力而为,来自 Agent)
	if data, err := st.Agent.DockerVersion(ctx); err == nil {
		if v, ok := data["version"].(string); ok && v != "" {
			info["docker_version"] = v
		}
	}
	return info
}

// MonitorSnapshot 监控快照(/system/monitor;Agent 采样,面板组装)
func MonitorSnapshot(st *state.AppState) map[string]any {
	out := map[string]any{
		"cpu_pct":  0.0,
		"mem":      map[string]any{},
		"load":     nil,
		"swap":     map[string]any{},
		"disk":     map[string]any{},
		"panel":    map[string]any{},
		"publicIP": map[string]any{"ipv4": "", "ipv6": ""},
		"net":      map[string]any{"rx_rate": 0.0, "tx_rate": 0.0, "rx_total": 0, "tx_total": 0},
		"io":       map[string]any{"read_rate": 0.0, "write_rate": 0.0, "read_total": 0, "write_total": 0},
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if st.Agent == nil {
		return out
	}
	// 宿主 CPU/内存/负载/交换/磁盘(Agent /v1/host/monitor)
	if data, err := st.Agent.HostMonitor(ctx); err == nil {
		for _, k := range []string{"cpu_pct", "mem", "load", "swap", "disk", "server_time"} {
			if v, ok := data[k]; ok {
				out[k] = v
			}
		}
	}
	// 容器网络 / IO 速率 + 累计值(Agent /v1/docker/io,8s 缓存差分)
	if data, err := st.Agent.DockerIO(ctx); err == nil {
		if netMap, ok := data["net"].(map[string]any); ok {
			out["net"] = netMap
		}
		if ioMap, ok := data["io"].(map[string]any); ok {
			out["io"] = ioMap
		}
	}

	// 面板自身进程(本地)
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	out["panel"] = map[string]any{"mem": ms.Sys, "threads": runtime.NumGoroutine()}

	// 公网 IP(本地探测)
	pub4, pub6 := PublicIPs()
	out["publicIP"] = map[string]any{"ipv4": pub4, "ipv6": pub6}
	return out
}

// ---------------- 镜像加速 (daemon.json registry-mirrors,Agent 管理宿主) ----------------

// RegistryMirrors 读取当前镜像加速配置
func RegistryMirrors(st *state.AppState) (mirrors []string, path string, exists bool) {
	if st.Agent == nil {
		return []string{}, "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	data, err := st.Agent.RegistryMirrors(ctx)
	if err != nil {
		return []string{}, "", false
	}
	path, _ = data["path"].(string)
	exists, _ = data["exists"].(bool)
	mirrors = []string{}
	if arr, ok := data["mirrors"].([]any); ok {
		for _, m := range arr {
			if s, ok := m.(string); ok {
				mirrors = append(mirrors, s)
			}
		}
	}
	return mirrors, path, exists
}

// SaveRegistryMirrors 写入镜像加速配置
func SaveRegistryMirrors(st *state.AppState, mirrors []string) error {
	if st.Agent == nil {
		return NewApiError(502, "agent.unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := st.Agent.SaveRegistryMirrors(ctx, mirrors); err != nil {
		return agentToApiError(err)
	}
	return nil
}

// RestartDocker 重启 Docker 服务使镜像加速生效
func RestartDocker(st *state.AppState) error {
	if st.Agent == nil {
		return NewApiError(502, "agent.unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := st.Agent.DockerServiceAction(ctx, "restart"); err != nil {
		return agentToApiError(err)
	}
	return nil
}
