package api

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// ---------------- 在线更新 ----------------

// AppVersion 面板当前版本(发版时与 main.go 的 Version 同步;CI 以 main.go 字面量为准)
const AppVersion = "1.0.0"

const (
	updateGitHubURL  = "https://api.github.com/repos/MinimaxFlora/Docker_Manager_Go/releases/latest"
	updateCheckTTL   = 10 * time.Minute
	composeHelperImg = "docker/compose:latest"
	updateHelperName = "dm-update-helper"
)

// updateCheckURL 检测接口地址;DM_UPDATE_API 环境变量可覆盖(本地演示/测试用,生产默认 GitHub 官方)
func updateCheckURL() string {
	if u := os.Getenv("DM_UPDATE_API"); u != "" {
		return u
	}
	return updateGitHubURL
}

type updateRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	HTMLURL     string `json:"html_url"`
	Prerelease  bool   `json:"prerelease"`
}

type updateInfo struct {
	Current   string         `json:"current"`
	Latest    string         `json:"latest"`
	HasUpdate bool           `json:"has_update"`
	Release   *updateRelease `json:"release,omitempty"`
	Error     string         `json:"error,omitempty"`
}

var (
	updateMu        sync.Mutex
	updateCheckedAt time.Time
	updateCache     *updateInfo
	applyMu         sync.Mutex
)

// compareVersions 比较版本号("1.0.0" / "v1.0.1"):a<b → -1,a==b → 0,a>b → 1
func compareVersions(a, b string) int {
	na, nb := versionNums(a), versionNums(b)
	n := len(na)
	if len(nb) > n {
		n = len(nb)
	}
	for i := 0; i < n; i++ {
		x, y := 0, 0
		if i < len(na) {
			x = na[i]
		}
		if i < len(nb) {
			y = nb[i]
		}
		if x < y {
			return -1
		}
		if x > y {
			return 1
		}
	}
	return 0
}

func versionNums(s string) []int {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 { // 截断预发布/构建后缀
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	out := make([]int, 0, len(parts))
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			n = 0
		}
		out = append(out, n)
	}
	return out
}

// updateCheck GET /api/update/check — 检测 GitHub 最新 Release(结果缓存 10 分钟,防 GitHub API 限流)
func updateCheck(c *gin.Context, st *state.AppState) error {
	updateMu.Lock()
	defer updateMu.Unlock()
	if updateCache != nil && time.Since(updateCheckedAt) < updateCheckTTL {
		c.JSON(200, updateCache)
		return nil
	}

	info := &updateInfo{Current: AppVersion}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateCheckURL(), nil)
	if err == nil {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "docker-manager-go/"+AppVersion)
		if resp, err := (&http.Client{}).Do(req); err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var rel updateRelease
				if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil || rel.TagName == "" {
					info.Error = "invalid release payload"
				} else {
					latest := strings.TrimPrefix(rel.TagName, "v")
					info.Latest = latest
					info.Release = &rel
					info.HasUpdate = compareVersions(AppVersion, latest) < 0
				}
			} else {
				info.Error = fmt.Sprintf("github api http %d", resp.StatusCode)
			}
		} else {
			info.Error = "github api unreachable"
		}
	} else {
		info.Error = "bad request"
	}

	// 仅成功结果缓存 10 分钟;失败不缓存,下次请求立即重试
	if info.Error == "" {
		updateCache = info
		updateCheckedAt = time.Now()
	}
	c.JSON(200, info)
	return nil
}

// updateApply POST /api/update/apply — 一键更新,按部署模式分流:
//   - binary(systemd 直跑):下载 release 二进制资产 → 原子替换自身 → 延迟重启服务
//   - compose(容器内):启动独立 docker/compose 容器代跑 pull + recreate
func updateApply(c *gin.Context, st *state.AppState) error {
	applyMu.Lock()
	defer applyMu.Unlock()
	if deploymentMode() == "binary" {
		return applyBinaryUpdate(c)
	}
	return applyComposeUpdate(c, st)
}

// deploymentMode 部署模式:DM_DEPLOY_MODE 环境变量可覆盖(演示/测试);
// 生产自动判断:容器内(cgroup 含 docker 段)→ compose,否则 binary
func deploymentMode() string {
	if m := os.Getenv("DM_DEPLOY_MODE"); m == "compose" || m == "binary" {
		return m
	}
	if selfContainerID() != "" {
		return "compose"
	}
	return "binary"
}

// applyComposeUpdate compose 部署:独立 compose 容器代跑(面板容器重建期间执行者不受影响)
func applyComposeUpdate(c *gin.Context, st *state.AppState) error {
	dir, err := findComposeDir(st)
	if err != nil {
		return NewApiError(400, err.Error())
	}

	// 用独立 context(5 分钟):拉镜像/建容器不随请求断开而中断
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. 拉取 compose 辅助镜像(读完进度流即等待拉取完成)
	pullRes, err := st.Docker.ImagePull(ctx, composeHelperImg, client.ImagePullOptions{})
	if err != nil {
		return NewApiError(502, "拉取更新辅助镜像失败: "+err.Error())
	}
	for range pullRes.JSONMessages(ctx) {
	}
	_ = pullRes.Close()

	// 2. 清理上次更新遗留的 helper 容器(幂等)
	_, _ = st.Docker.ContainerRemove(ctx, updateHelperName, client.ContainerRemoveOptions{Force: true})

	// 3. 创建并启动 helper:挂载 docker.sock + compose 文件目录(宿主路径由 daemon 解释)
	_, err = st.Docker.ContainerCreate(ctx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image:      composeHelperImg,
			Cmd:        []string{"-f", "/work/docker-compose.yml", "up", "-d", "--force-recreate", "--pull", "always"},
			WorkingDir: "/work",
		},
		HostConfig: &container.HostConfig{
			Binds: []string{
				"/var/run/docker.sock:/var/run/docker.sock",
				dir + ":/work:ro",
			},
			AutoRemove: true,
		},
		Name: updateHelperName,
	})
	if err != nil {
		return NewApiError(502, "创建更新容器失败: "+err.Error())
	}
	if _, err := st.Docker.ContainerStart(ctx, updateHelperName, client.ContainerStartOptions{}); err != nil {
		return NewApiError(502, "启动更新容器失败: "+err.Error())
	}
	c.JSON(200, gin.H{"ok": true, "message": "update.started"})
	return nil
}

// applyBinaryUpdate binary(systemd)部署:下载 GitHub Release 二进制资产,
// 校验后原子替换自身(/proc/self/exe),1.5 秒后 systemctl restart 服务。
// 响应先返回,重启由 goroutine 延迟触发,避免进程被杀导致响应丢失。
func applyBinaryUpdate(c *gin.Context) error {
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		return NewApiError(400, "不支持的架构: "+arch)
	}
	pkg := fmt.Sprintf("docker-manager-go-linux-%s.tar.gz", arch)
	url := "https://github.com/MinimaxFlora/Docker_Manager_Go/releases/latest/download/" + pkg

	// 1. 下载资产(120s 超时,避免挂死)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return NewApiError(502, "下载更新包失败: "+err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return NewApiError(502, fmt.Sprintf("下载更新包失败: HTTP %d", resp.StatusCode))
	}
	tmpFile, err := os.CreateTemp("", "dm-update-*.tar.gz")
	if err != nil {
		return NewApiError(500, "创建临时文件失败: "+err.Error())
	}
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return NewApiError(502, "下载中断: "+err.Error())
	}
	tmpFile.Close()

	// 2. 解压出新的可执行文件
	newBin, err := extractBinary(tmpFile.Name())
	os.Remove(tmpFile.Name())
	if err != nil {
		return NewApiError(500, "解压更新包失败: "+err.Error())
	}
	defer os.Remove(newBin)

	// 3. 定位当前二进制并原子替换(先备份,失败可回滚)
	exe, err := os.Readlink("/proc/self/exe")
	if err != nil || exe == "" {
		return NewApiError(500, "无法定位当前二进制(/proc/self/exe)")
	}
	// 新二进制先复制到目标同目录(/tmp 常为 tmpfs,跨文件系统 rename 会 EXDEV),再原子替换
	staged := exe + ".new"
	if err := copyFile(newBin, staged); err != nil {
		return NewApiError(500, "暂存新二进制失败: "+err.Error())
	}
	defer os.Remove(staged)
	if err := os.Chmod(staged, 0o755); err != nil {
		return NewApiError(500, "设置权限失败: "+err.Error())
	}
	_ = os.Remove(exe + ".old")
	if err := os.Rename(exe, exe+".old"); err != nil {
		return NewApiError(500, "备份旧二进制失败: "+err.Error())
	}
	if err := os.Rename(staged, exe); err != nil {
		_ = os.Rename(exe+".old", exe) // 回滚
		return NewApiError(500, "替换二进制失败: "+err.Error())
	}
	// 保留 exe.old 作为备份(下次更新覆盖):新二进制若无法启动,可手动恢复
	log.Printf("update: 二进制已替换 %s (备份 %s.old),即将重启服务", exe, exe)

	// 4. 延迟重启服务(响应先发出)
	go func() {
		time.Sleep(1500 * time.Millisecond)
		if err := exec.Command("systemctl", "restart", "docker-manager").Run(); err != nil {
			log.Printf("update: systemctl restart docker-manager 失败: %v (可手动执行恢复)", err)
		}
	}()
	c.JSON(200, gin.H{"ok": true, "message": "update.started"})
	return nil
}

// copyFile 复制文件内容与权限
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dst)
		return err
	}
	return out.Close()
}

// extractBinary 从发布 tar.gz 中解压出 docker-manager-go 可执行文件到临时目录
func extractBinary(tarGz string) (string, error) {
	f, err := os.Open(tarGz)
	if err != nil {
		return "", err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if hdr.Typeflag != tar.TypeReg || !strings.HasSuffix(hdr.Name, "docker-manager-go") {
			continue
		}
		out, err := os.CreateTemp("", "dm-bin-*")
		if err != nil {
			return "", err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			os.Remove(out.Name())
			return "", err
		}
		out.Close()
		return out.Name(), nil
	}
	return "", fmt.Errorf("压缩包中未找到 docker-manager-go")
}

// findComposeDir 探测宿主 docker-compose.yml 所在目录(**宿主路径**,供 bind 挂载):
//  1. 从自身容器的 /data 挂载反推宿主 install 目录(install.sh: data 在 install 下)
//  2. 默认安装目录 /opt/docker-manager(经 /host 只读挂载,去掉 /host 前缀还原宿主路径)
func findComposeDir(st *state.AppState) (string, error) {
	var candidates []string
	if id := selfContainerID(); id != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if inj, err := st.Docker.ContainerInspect(ctx, id, client.ContainerInspectOptions{}); err == nil {
			for _, m := range inj.Container.Mounts {
				if m.Destination == "/data" && m.Source != "" {
					candidates = append(candidates, filepath.Join(filepath.Dir(m.Source), "docker-compose.yml"))
				}
			}
		}
	}
	candidates = append(candidates, "/host/opt/docker-manager/docker-compose.yml")
	for _, f := range candidates {
		if _, err := os.Stat(f); err == nil {
			// /host 挂载 = 宿主根,还原为宿主路径后才能作为 bind 挂载源传给 daemon
			return filepath.Dir(strings.TrimPrefix(f, "/host/")), nil
		}
	}
	return "", fmt.Errorf("未找到 docker-compose.yml(已探测: %s)。请在宿主机执行 install.sh update 手动更新", strings.Join(candidates, "、"))
}

// selfContainerID 从 /proc/self/cgroup 解析当前容器 ID
func selfContainerID() string {
	raw, err := os.ReadFile("/proc/self/cgroup")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if i := strings.LastIndex(line, "/"); i >= 0 {
			id := strings.TrimSpace(line[i+1:])
			if len(id) == 64 && isHexString(id) {
				return id
			}
		}
	}
	return ""
}

func isHexString(s string) bool {
	for _, r := range s {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return false
		}
	}
	return true
}
