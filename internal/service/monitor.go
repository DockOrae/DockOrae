package service

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/DockOrae/DockOrae/internal/docker"
	"github.com/DockOrae/DockOrae/internal/state"
)

// ---------------- 宿主机信息 (/proc) ----------------

func ReadOsName() string {
	if s, err := os.ReadFile("/etc/os-release"); err == nil {
		for _, line := range strings.Split(string(s), "\n") {
			if strings.HasPrefix(line, "PRETTY_NAME=") {
				return strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), "\"")
			}
		}
	}
	if s, err := os.ReadFile("/proc/version"); err == nil {
		return firstWords(string(s), 3)
	}
	return "unknown"
}

func ReadKernel() string {
	if s, err := os.ReadFile("/proc/version"); err == nil {
		return firstWords(string(s), 3)
	}
	return "unknown"
}

func firstWords(s string, n int) string {
	fields := strings.Fields(s)
	if len(fields) > n {
		fields = fields[:n]
	}
	return strings.Join(fields, " ")
}

func ReadUptime() (uint64, bool) {
	if s, err := os.ReadFile("/proc/uptime"); err == nil {
		fields := strings.Fields(string(s))
		if len(fields) > 0 {
			if f, err := strconv.ParseFloat(fields[0], 64); err == nil {
				return uint64(f), true
			}
		}
	}
	return 0, false
}

func ReadLoadavg() ([3]float64, bool) {
	var out [3]float64
	s, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return out, false
	}
	fields := strings.Fields(string(s))
	if len(fields) < 3 {
		return out, false
	}
	for i := 0; i < 3; i++ {
		v, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			return out, false
		}
		out[i] = v
	}
	return out, true
}

// ReadCpuinfo (CPU 型号, 核心数)
func ReadCpuinfo() (string, int) {
	model := ""
	cores := 0
	s, err := os.ReadFile("/proc/cpuinfo")
	if err != nil {
		return model, cores
	}
	for _, line := range strings.Split(string(s), "\n") {
		if strings.HasPrefix(line, "model name") {
			if parts := strings.SplitN(line, ":", 2); len(parts) == 2 {
				model = strings.TrimSpace(parts[1])
			}
		}
		if strings.HasPrefix(line, "processor") {
			cores++
		}
	}
	return model, cores
}

// ReadMeminfo (total_kb, available_kb)
func ReadMeminfo() (uint64, uint64, bool) {
	s, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0, 0, false
	}
	var total, avail uint64
	for _, line := range strings.Split(string(s), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch {
		case fields[0] == "MemTotal:":
			total, _ = strconv.ParseUint(fields[1], 10, 64)
		case fields[0] == "MemAvailable:":
			avail, _ = strconv.ParseUint(fields[1], 10, 64)
		}
	}
	return total, avail, true
}

// ReadCpuStat (idle, total) CPU 时钟
func ReadCpuStat() (uint64, uint64, bool) {
	s, err := os.ReadFile("/proc/stat")
	if err != nil {
		return 0, 0, false
	}
	line := strings.SplitN(string(s), "\n", 2)[0]
	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0, 0, false
	}
	var parts []uint64
	for _, f := range fields[1:] {
		v, err := strconv.ParseUint(f, 10, 64)
		if err != nil {
			return 0, 0, false
		}
		parts = append(parts, v)
	}
	var total uint64
	for _, v := range parts {
		total += v
	}
	idle := parts[3]
	if len(parts) > 4 {
		idle += parts[4]
	}
	return idle, total, true
}

// ---------------- 公网 IP(仿 3x-ui:多服务列表 + 5 分钟缓存) ----------------

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

// ---------------- 容器流量 / 磁盘 IO 速率 ----------------

// 前端每 3 秒轮询 /system/monitor,若每轮都对全部运行中容器打 stats,Docker daemon 会被持续压榨
// 并拖慢列表接口。因此 8 秒采样一次累计值,内部差分后直接返回"每秒速率(B/s)"。
var (
	netIOCacheMu   sync.Mutex
	netIOCacheAt   time.Time
	netIOCachePrev time.Time
	netIOCacheRx   uint64
	netIOCacheTx   uint64
	netIOCacheRd   uint64
	netIOCacheWr   uint64
	netRateRx      float64
	netRateTx      float64
	netRateRd      float64
	netRateWr      float64
)

const netIOCacheTTL = 8 * time.Second

// ContainerNetIO 返回所有运行中容器的网络收发/磁盘读写速率(B/s)与最近采样累计值
func ContainerNetIO(st *state.AppState) (rxRate, txRate, rdRate, wrRate float64, curRx, curTx, curRd, curWr uint64) {
	netIOCacheMu.Lock()
	defer netIOCacheMu.Unlock()
	if !netIOCacheAt.IsZero() && time.Since(netIOCacheAt) < netIOCacheTTL {
		return netRateRx, netRateTx, netRateRd, netRateWr, netIOCacheRx, netIOCacheTx, netIOCacheRd, netIOCacheWr
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	curRx, curTx, curRd, curWr, ok := sampleContainerIO(st, ctx)
	now := time.Now()
	if ok && !netIOCachePrev.IsZero() {
		if dt := now.Sub(netIOCachePrev).Seconds(); dt > 0 {
			netRateRx = rateDelta(curRx, netIOCacheRx, dt)
			netRateTx = rateDelta(curTx, netIOCacheTx, dt)
			netRateRd = rateDelta(curRd, netIOCacheRd, dt)
			netRateWr = rateDelta(curWr, netIOCacheWr, dt)
		}
	}
	// 采样成功才更新差分基准;失败保留旧基准并返回缓存累计值(避免前端总流量卡片闪 0)
	if ok {
		netIOCacheRx, netIOCacheTx, netIOCacheRd, netIOCacheWr = curRx, curTx, curRd, curWr
		netIOCachePrev = now
	} else {
		curRx, curTx, curRd, curWr = netIOCacheRx, netIOCacheTx, netIOCacheRd, netIOCacheWr
	}
	netIOCacheAt = now
	return netRateRx, netRateTx, netRateRd, netRateWr, curRx, curTx, curRd, curWr
}

// rateDelta 累计值差速:cur < prev(计数器重置/回绕)时返回 0
func rateDelta(cur, prev uint64, dt float64) float64 {
	if cur < prev || dt <= 0 {
		return 0
	}
	return float64(cur-prev) / dt
}

// sampleContainerIO 采样所有运行中容器的网络收发与磁盘读写累计值
func sampleContainerIO(st *state.AppState, ctx context.Context) (rx, tx, rd, wr uint64, ok bool) {
	items, err := docker.ListContainers(st.Docker, ctx, client.ContainerListOptions{})
	if err != nil {
		return 0, 0, 0, 0, false
	}
	for _, c := range items {
		if string(c.State) != "running" {
			continue
		}
		statsRes, err := docker.ContainerStats(st.Docker, ctx, c.ID, client.ContainerStatsOptions{})
		if err != nil {
			continue
		}
		var s container.StatsResponse
		decErr := json.NewDecoder(statsRes.Body).Decode(&s)
		statsRes.Body.Close()
		if decErr != nil {
			continue
		}
		for _, n := range s.Networks {
			rx += n.RxBytes
			tx += n.TxBytes
		}
		for _, r := range s.BlkioStats.IoServiceBytesRecursive {
			switch r.Op {
			case "read":
				rd += r.Value
			case "write":
				wr += r.Value
			}
		}
	}
	return rx, tx, rd, wr, true
}

// ---------------- 宿主 / 监控快照 ----------------

// HostInfo 宿主机信息(/system/host)
func HostInfo(st *state.AppState) map[string]any {
	cpuModel, cores := ReadCpuinfo()
	totalMem, _, _ := ReadMeminfo()
	uptime, _ := ReadUptime()
	osName := ReadOsName()
	kernel := ReadKernel()

	// Docker 版本(尽力而为)
	dockerVer := ""
	if info, err := st.Docker.ServerVersion(context.Background(), client.ServerVersionOptions{}); err == nil {
		dockerVer = info.Version
	}

	arch := runtime.GOARCH
	if arch == "amd64" {
		arch = "x86_64"
	} else if arch == "arm64" {
		arch = "aarch64"
	}

	return map[string]any{
		"hostname":       hostnameOrUnknown(),
		"os":             osName,
		"kernel":         kernel,
		"arch":           arch,
		"cpu_model":      cpuModel,
		"cpu_cores":      cores,
		"mem_total":      totalMem * 1024,
		"uptime":         uptime,
		"docker_version": dockerVer,
		"server_time":    time.Now().Unix(),
	}
}

func hostnameOrUnknown() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

// MonitorSnapshot 监控快照(/system/monitor)
func MonitorSnapshot(st *state.AppState) map[string]any {
	cpuPct := 0.0
	if idle, total, ok := ReadCpuStat(); ok {
		cpuPct = st.SampleCPU(idle, total)
	}

	mem := map[string]any{}
	if total, avail, ok := ReadMeminfo(); ok && total > 0 {
		used := total - avail
		mem = map[string]any{"total": total * 1024, "used": used * 1024, "pct": float64(used) / float64(total) * 100.0}
	}

	// 负载
	var load any
	if l, ok := ReadLoadavg(); ok {
		load = l
	}

	// 交换空间
	var swap any
	if t, u, ok := ReadSwap(); ok {
		pct := 0.0
		if t > 0 {
			pct = float64(u) / float64(t) * 100.0
		}
		swap = map[string]any{"total": t, "used": u, "pct": pct}
	}

	// 磁盘
	var disk any
	if total, used, ok := ReadDisk(); ok {
		pct := 0.0
		if total > 0 {
			pct = float64(used) / float64(total) * 100.0
		}
		disk = map[string]any{"total": total, "used": used, "pct": pct}
	}

	// 面板自身进程
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	panel := map[string]any{"mem": ms.Sys, "threads": runtime.NumGoroutine()}

	// 公网 IP
	pub4, pub6 := PublicIPs()

	// 容器网络 / IO 速率 + 累计值
	rxRate, txRate, rdRate, wrRate, rxTotal, txTotal, rdTotal, wrTotal := ContainerNetIO(st)

	return map[string]any{
		"cpu_pct":  cpuPct,
		"mem":      mem,
		"load":     load,
		"swap":     swap,
		"disk":     disk,
		"panel":    panel,
		"publicIP": map[string]any{"ipv4": pub4, "ipv6": pub6},
		"net":      map[string]any{"rx_rate": rxRate, "tx_rate": txRate, "rx_total": rxTotal, "tx_total": txTotal},
		"io":       map[string]any{"read_rate": rdRate, "write_rate": wrRate, "read_total": rdTotal, "write_total": wrTotal},
	}
}

// ---------------- 镜像加速 (daemon.json registry-mirrors) ----------------

// DaemonJSONPath 定位 daemon.json:环境变量 → 容器挂载的宿主路径 → 默认路径
func DaemonJSONPath() string {
	if p := os.Getenv("DOCKER_DAEMON_JSON"); p != "" {
		return p
	}
	for _, p := range []string{"/host/etc/docker/daemon.json", "/etc/docker/daemon.json"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	if _, err := os.Stat("/host/etc/docker"); err == nil {
		return "/host/etc/docker/daemon.json"
	}
	return "/etc/docker/daemon.json"
}

// RegistryMirrors 读取当前镜像加速配置
func RegistryMirrors() (mirrors []string, path string, exists bool) {
	path = DaemonJSONPath()
	mirrors = []string{}
	if _, err := os.Stat(path); err == nil {
		exists = true
		if raw, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(raw)) != "" {
			var v map[string]any
			if json.Unmarshal(raw, &v) == nil {
				if arr, ok := v["registry-mirrors"].([]any); ok {
					for _, m := range arr {
						if s, ok := m.(string); ok {
							mirrors = append(mirrors, s)
						}
					}
				}
			}
		}
	}
	return mirrors, path, exists
}

// SaveRegistryMirrors 写入镜像加速配置
func SaveRegistryMirrors(mirrors []string) error {
	path := DaemonJSONPath()
	cfg := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(raw)) != "" {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return BadRequest("mirror.parseFailed: " + err.Error())
		}
	}
	if len(mirrors) == 0 {
		delete(cfg, "registry-mirrors")
	} else {
		cfg["registry-mirrors"] = mirrors
	}
	if dir := path[:strings.LastIndex(path, "/")]; dir != "" {
		_ = os.MkdirAll(dir, 0o755)
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return BadRequest("mirror.parseFailed: " + err.Error())
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return BadRequest("写入失败: " + err.Error())
	}
	return nil
}

// RestartDocker 重启 Docker 服务使镜像加速生效(尽力而为)
func RestartDocker() error {
	attempts := [][]string{
		{"nsenter", "-t", "1", "-m", "--", "systemctl", "restart", "docker"},
		{"systemctl", "restart", "docker"},
	}
	for _, cmd := range attempts {
		if err := exec.Command(cmd[0], cmd[1:]...).Run(); err == nil {
			return nil
		}
	}
	return BadRequest("mirror.restartFailed")
}
