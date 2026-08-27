package api

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ---------------- 宿主机信息 (/proc) ----------------

func readOsName() string {
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

func readKernel() string {
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

func readUptime() (uint64, bool) {
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

func readLoadavg() ([3]float64, bool) {
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

// readCpuinfo (CPU 型号, 核心数)
func readCpuinfo() (string, int) {
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

// readMeminfo (total_kb, available_kb)
func readMeminfo() (uint64, uint64, bool) {
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

// readCpuStat (idle, total) CPU 时钟
func readCpuStat() (uint64, uint64, bool) {
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

// ---------------- 容器流量 / 磁盘 IO 汇总 ----------------

// containerNetIO 汇总所有运行中容器的网络收发与磁盘读写累计值 -> (rx, tx, read, write)
func containerNetIO(st *state.AppState) (rx, tx, rd, wr uint64) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	res, err := st.Docker.ContainerList(ctx, client.ContainerListOptions{})
	if err != nil {
		return 0, 0, 0, 0
	}
	for _, c := range res.Items {
		if string(c.State) != "running" {
			continue
		}
		statsRes, err := st.Docker.ContainerStats(ctx, c.ID, client.ContainerStatsOptions{})
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
	return rx, tx, rd, wr
}

// ---------------- 镜像加速 (daemon.json registry-mirrors) ----------------

// daemonJSONPath 定位 daemon.json:优先环境变量 → 容器挂载的宿主路径 → 默认路径
func daemonJSONPath() string {
	if p := os.Getenv("DOCKER_DAEMON_JSON"); p != "" {
		return p
	}
	for _, p := range []string{"/host/etc/docker/daemon.json", "/etc/docker/daemon.json"} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	// 都不存在时,优先可写路径(/host 挂载优先)
	if _, err := os.Stat("/host/etc/docker"); err == nil {
		return "/host/etc/docker/daemon.json"
	}
	return "/etc/docker/daemon.json"
}

func monitorRegistryMirrors(c *gin.Context, st *state.AppState) error {
	path := daemonJSONPath()
	mirrors := []string{}
	exists := false
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
	c.JSON(200, gin.H{"mirrors": mirrors, "path": path, "exists": exists})
	return nil
}

func monitorSaveRegistryMirrors(c *gin.Context, st *state.AppState) error {
	var payload struct {
		Mirrors []string `json:"mirrors"`
	}
	if err := c.ShouldBindJSON(&payload); err != nil {
		return BadRequest("err.requestFailed")
	}
	path := daemonJSONPath()

	cfg := map[string]any{}
	if raw, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(raw)) != "" {
		if err := json.Unmarshal(raw, &cfg); err != nil {
			return BadRequest("mirror.parseFailed: " + err.Error())
		}
	}
	if len(payload.Mirrors) == 0 {
		delete(cfg, "registry-mirrors")
	} else {
		ms := make([]string, 0, len(payload.Mirrors))
		for _, m := range payload.Mirrors {
			ms = append(ms, m)
		}
		cfg["registry-mirrors"] = ms
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
	c.JSON(200, gin.H{"ok": true, "path": path})
	return nil
}

// ---------------- 接口 ----------------

// monitorHost 系统信息(主机名/系统/内核/运行时间/负载/CPU/内存等)
func monitorHost(c *gin.Context, st *state.AppState) error {
	cpuModel, cpuCores := readCpuinfo()
	memTotalKb, _, _ := readMeminfo()
	load, _ := readLoadavg()
	uptime, _ := readUptime()

	// Docker 信息补充(宿主机 OS / 内核 / 架构,容器内读不到时兜底)
	arch := ""
	if info, err := st.Docker.Info(c.Request.Context(), client.InfoOptions{}); err == nil {
		arch = info.Info.Architecture
	}
	hostname, _ := os.Hostname()

	c.JSON(200, gin.H{
		"hostname":    hostname,
		"os":          readOsName(),
		"kernel":      readKernel(),
		"uptime":      uptime,
		"load":        load,
		"cpu_model":   cpuModel,
		"cpu_cores":   cpuCores,
		"mem_total":   memTotalKb * 1024,
		"arch":        arch,
		"server_time": time.Now().Unix(),
	})
	return nil
}

// monitorMonitor 实时监控:CPU/内存/负载/磁盘 + 容器网络/IO 累计值
func monitorMonitor(c *gin.Context, st *state.AppState) error {
	// CPU 使用率(与上次采样差值)
	cpuPct := 0.0
	if idle, total, ok := readCpuStat(); ok {
		cpuPct = st.SampleCPU(idle, total)
	}

	// 内存
	var mem any
	if totalKb, availKb, ok := readMeminfo(); ok {
		total := totalKb * 1024
		used := total - availKb*1024
		pct := 0.0
		if total > 0 {
			pct = float64(used) / float64(total) * 100.0
		}
		mem = gin.H{"total": total, "used": used, "pct": pct}
	}

	// 负载
	var load any
	if l, ok := readLoadavg(); ok {
		load = l
	}

	// 磁盘
	var disk any
	if total, used, ok := readDisk(); ok {
		pct := 0.0
		if total > 0 {
			pct = float64(used) / float64(total) * 100.0
		}
		disk = gin.H{"total": total, "used": used, "pct": pct}
	}

	// 容器网络 / IO 累计值(前端做差值算速率)
	rx, tx, rd, wr := containerNetIO(st)

	c.JSON(200, gin.H{
		"cpu_pct": cpuPct,
		"mem":     mem,
		"load":    load,
		"disk":    disk,
		"net":     gin.H{"rx": rx, "tx": tx},
		"io":      gin.H{"read": rd, "write": wr},
	})
	return nil
}

// monitorRestartDocker 重启 Docker 服务使镜像加速生效(尽力而为)
func monitorRestartDocker(c *gin.Context, st *state.AppState) error {
	attempts := [][]string{
		{"nsenter", "-t", "1", "-m", "--", "systemctl", "restart", "docker"},
		{"systemctl", "restart", "docker"},
	}
	for _, cmd := range attempts {
		out := exec.Command(cmd[0], cmd[1:]...)
		if err := out.Run(); err == nil {
			c.JSON(200, gin.H{"ok": true})
			return nil
		}
	}
	return BadRequest("mirror.restartFailed")
}
