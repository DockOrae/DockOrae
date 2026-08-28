package service

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
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

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/docker"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// AppVersion 面板当前版本,构建时由 ldflags 注入(与 main.Version 同步,发版打 tag 即可):
//
//	-X github.com/MinimaxFlora/Docker_Manager_Go/internal/service.AppVersion=v1.0.3
//
// 未注入(本地开发)时默认 dev。使用 Makefile 构建时自动注入。
var AppVersion = "dev"

// ---------- 更新状态(异步化 + 前端轮询) ----------

// UpdatePhase 更新进行阶段(前端按 phase 显示进度文案)
type UpdatePhase string

const (
	PhaseIdle        UpdatePhase = "idle"
	PhaseDownloading UpdatePhase = "downloading" // binary:下载 release 包
	PhaseExtracting  UpdatePhase = "extracting"  // binary:解压新二进制
	PhaseReplacing   UpdatePhase = "replacing"   // binary:原子替换自身
	PhaseRestarting  UpdatePhase = "restarting"  // binary:延迟重启服务
	PhasePulling     UpdatePhase = "pulling"     // compose:拉取 helper 镜像
	PhaseHelper      UpdatePhase = "helper"      // compose:启动 helper 容器
	PhaseDone        UpdatePhase = "done"        // compose:helper 已接管
	PhaseFailed      UpdatePhase = "failed"
)

// UpdateStatus 更新进度(binary/compose 通用;内存态,面板重启后自然消失——
// 重启本身即更新成功的标志,前端在轮询失败后改查版本接口确认新版本上线)
type UpdateStatus struct {
	Running    bool        `json:"running"`
	Phase      UpdatePhase `json:"phase"`
	Error      string      `json:"error"`
	StartedAt  time.Time   `json:"started_at"`
	FinishedAt time.Time   `json:"finished_at"`
}

var (
	updateStatusMu sync.Mutex
	updateStatus   UpdateStatus
)

// setUpdatePhase 推进更新阶段
func setUpdatePhase(p UpdatePhase) {
	updateStatusMu.Lock()
	defer updateStatusMu.Unlock()
	updateStatus.Running = p != PhaseIdle && p != PhaseDone && p != PhaseFailed
	updateStatus.Phase = p
	updateStatus.Error = ""
	if p == PhaseDone || p == PhaseFailed {
		updateStatus.FinishedAt = time.Now()
	}
}

// failUpdate 记录失败状态并返回 error(异步后错误不再经 HTTP 状态码回传)
func failUpdate(format string, args ...any) error {
	msg := fmt.Sprintf(format, args...)
	updateStatusMu.Lock()
	updateStatus.Running = false
	updateStatus.Phase = PhaseFailed
	updateStatus.Error = msg
	updateStatus.FinishedAt = time.Now()
	updateStatusMu.Unlock()
	log.Printf("update: %s", msg)
	return errors.New(msg)
}

// GetUpdateStatus 返回当前更新状态副本(供 /update/status 轮询)
func GetUpdateStatus() UpdateStatus {
	updateStatusMu.Lock()
	defer updateStatusMu.Unlock()
	return updateStatus
}

const (
	updateGitHubURL  = "https://api.github.com/repos/MinimaxFlora/Docker_Manager_Go/releases/latest"
	updateCheckTTL   = 10 * time.Minute
	composeHelperImg = "docker/compose:latest"
	updateHelperName = "dm-update-helper"
)

// UpdateCheckURL 检测接口地址;DM_UPDATE_API 环境变量可覆盖(本地演示/测试用,生产默认 GitHub 官方)
func UpdateCheckURL() string {
	if u := os.Getenv("DM_UPDATE_API"); u != "" {
		return u
	}
	return updateGitHubURL
}

var (
	updateMu        sync.Mutex
	updateCheckedAt time.Time
	updateCache     *model.UpdateInfo
)

// CompareVersions 比较版本号("1.0.0" / "v1.0.1"):a<b → -1,a==b → 0,a>b → 1
func CompareVersions(a, b string) int {
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

// UpdateCheck 检测 GitHub 最新 Release(结果缓存 10 分钟,防 GitHub API 限流;失败不缓存)
func UpdateCheck(st *state.AppState, ctx context.Context) (*model.UpdateInfo, error) {
	updateMu.Lock()
	defer updateMu.Unlock()
	if updateCache != nil && time.Since(updateCheckedAt) < updateCheckTTL {
		return updateCache, nil
	}

	info := &model.UpdateInfo{Current: AppVersion}
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(checkCtx, http.MethodGet, UpdateCheckURL(), nil)
	if err == nil {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("User-Agent", "docker-manager-go/"+AppVersion)
		if resp, err := (&http.Client{}).Do(req); err == nil {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				var rel model.UpdateRelease
				if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil || rel.TagName == "" {
					info.Error = "invalid release payload"
				} else {
					latest := strings.TrimPrefix(rel.TagName, "v")
					info.Latest = latest
					info.Release = &rel
					info.HasUpdate = CompareVersions(AppVersion, latest) < 0
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
	return info, nil
}

// UpdateApply 一键更新(异步):立即返回,goroutine 后台执行,进度经 GetUpdateStatus 轮询。
// 已在进行时返回 409;其余错误通过状态(PhaseFailed + Error)回传。
func UpdateApply(st *state.AppState, ctx context.Context) error {
	updateStatusMu.Lock()
	if updateStatus.Running {
		updateStatusMu.Unlock()
		return NewApiError(409, "更新已在进行中,请稍候")
	}
	updateStatus = UpdateStatus{Running: true, Phase: PhaseDownloading, StartedAt: time.Now()}
	updateStatusMu.Unlock()

	go func() {
		// 用独立 context:请求结束后 ctx 会被取消,后台任务不受影响
		var err error
		if DeploymentMode() == "binary" {
			err = applyBinaryUpdate(context.Background())
		} else {
			err = applyComposeUpdate(st, context.Background())
		}
		if err != nil {
			// 各阶段已通过 failUpdate 记录;兜底补记(防止漏网错误)
			updateStatusMu.Lock()
			if updateStatus.Phase != PhaseFailed {
				updateStatus.Running = false
				updateStatus.Phase = PhaseFailed
				updateStatus.Error = err.Error()
				updateStatus.FinishedAt = time.Now()
			}
			updateStatusMu.Unlock()
		}
	}()
	return nil
}

// DeploymentMode 部署模式:DM_DEPLOY_MODE 环境变量可覆盖(演示/测试);
// 生产自动判断:容器内(cgroup 含 docker 段)→ compose,否则 binary
func DeploymentMode() string {
	if m := os.Getenv("DM_DEPLOY_MODE"); m == "compose" || m == "binary" {
		return m
	}
	if SelfContainerID() != "" {
		return "compose"
	}
	return "binary"
}

// applyComposeUpdate compose 部署:独立 compose 容器代跑(面板容器重建期间执行者不受影响)
func applyComposeUpdate(st *state.AppState, ctx context.Context) error {
	dir, err := FindComposeDir(st)
	if err != nil {
		return failUpdate("%s", err.Error())
	}

	setUpdatePhase(PhasePulling)
	// 用独立 context(5 分钟):拉镜像/建容器不随请求断开而中断
	opCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 1. 拉取 compose 辅助镜像(读完进度流即等待拉取完成)
	pullRes, err := docker.PullImage(st.Docker, opCtx, composeHelperImg, client.ImagePullOptions{})
	if err != nil {
		return failUpdate("拉取更新辅助镜像失败: %s", err.Error())
	}
	for range pullRes.JSONMessages(opCtx) {
	}
	_ = pullRes.Close()

	// 2. 清理上次更新遗留的 helper 容器(幂等)
	_ = docker.RemoveContainer(st.Docker, opCtx, updateHelperName, client.ContainerRemoveOptions{Force: true})

	setUpdatePhase(PhaseHelper)
	// 3. 创建并启动 helper:挂载 docker.sock + compose 文件目录(宿主路径由 daemon 解释)
	_, err = docker.CreateContainer(st.Docker, opCtx, client.ContainerCreateOptions{
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
		return failUpdate("创建更新容器失败: %s", err.Error())
	}
	if err := docker.StartContainer(st.Docker, opCtx, updateHelperName); err != nil {
		return failUpdate("启动更新容器失败: %s", err.Error())
	}

	// helper 已接管更新(它会 pull 新镜像并 recreate 面板容器),标记完成供前端转轮询版本接口
	setUpdatePhase(PhaseDone)
	return nil
}

// applyBinaryUpdate binary(systemd)部署:下载 GitHub Release 二进制资产,
// 校验后原子替换自身(/proc/self/exe),1.5 秒后 systemctl restart 服务。
// 阶段进度经 setUpdatePhase 上报,错误经 failUpdate 记录。
func applyBinaryUpdate(ctx context.Context) error {
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		return failUpdate("不支持的架构: %s", arch)
	}
	pkg := fmt.Sprintf("docker-manager-go-linux-%s.tar.gz", arch)
	url := "https://github.com/MinimaxFlora/Docker_Manager_Go/releases/latest/download/" + pkg

	setUpdatePhase(PhaseDownloading)
	// 1. 下载资产(120s 超时,避免挂死)
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return failUpdate("下载更新包失败: %s", err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return failUpdate("下载更新包失败: HTTP %d", resp.StatusCode)
	}
	tmpFile, err := os.CreateTemp("", "dm-update-*.tar.gz")
	if err != nil {
		return failUpdate("创建临时文件失败: %s", err.Error())
	}
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return failUpdate("下载中断: %s", err.Error())
	}
	tmpFile.Close()

	// 2. 解压出新的可执行文件
	setUpdatePhase(PhaseExtracting)
	newBin, err := extractBinary(tmpFile.Name())
	os.Remove(tmpFile.Name())
	if err != nil {
		return failUpdate("解压更新包失败: %s", err.Error())
	}
	defer os.Remove(newBin)

	// 3. 定位当前二进制并原子替换(先备份,失败可回滚)
	setUpdatePhase(PhaseReplacing)
	exe, err := os.Readlink("/proc/self/exe")
	if err != nil || exe == "" {
		return failUpdate("无法定位当前二进制(/proc/self/exe)")
	}
	// 新二进制先复制到目标同目录(/tmp 常为 tmpfs,跨文件系统 rename 会 EXDEV),再原子替换
	staged := exe + ".new"
	if err := copyFile(newBin, staged); err != nil {
		return failUpdate("暂存新二进制失败: %s", err.Error())
	}
	defer os.Remove(staged)
	if err := os.Chmod(staged, 0o755); err != nil {
		return failUpdate("设置权限失败: %s", err.Error())
	}
	_ = os.Remove(exe + ".old")
	if err := os.Rename(exe, exe+".old"); err != nil {
		return failUpdate("备份旧二进制失败: %s", err.Error())
	}
	if err := os.Rename(staged, exe); err != nil {
		_ = os.Rename(exe+".old", exe) // 回滚
		return failUpdate("替换二进制失败: %s", err.Error())
	}
	// 保留 exe.old 作为备份(下次更新覆盖):新二进制若无法启动,可手动恢复
	log.Printf("update: 二进制已替换 %s (备份 %s.old),即将重启服务", exe, exe)

	// 4. 延迟重启服务(状态停在 restarting,进程退出后由前端轮询版本接口确认)
	setUpdatePhase(PhaseRestarting)
	go func() {
		time.Sleep(1500 * time.Millisecond)
		if err := exec.Command("systemctl", "restart", "docker-manager").Run(); err != nil {
			log.Printf("update: systemctl restart docker-manager 失败: %v (可手动执行恢复)", err)
		}
	}()
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

// FindComposeDir 探测宿主 docker-compose.yml 所在目录(**宿主路径**,供 bind 挂载):
//  1. 从自身容器的 /data 挂载反推宿主 install 目录(install.sh: data 在 install 下)
//  2. 默认安装目录 /opt/docker-manager(经 /host 只读挂载,去掉 /host 前缀还原宿主路径)
func FindComposeDir(st *state.AppState) (string, error) {
	var candidates []string
	if id := SelfContainerID(); id != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if inj, err := docker.InspectContainer(st.Docker, ctx, id); err == nil {
			for _, m := range inj.Mounts {
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

// SelfContainerID 从 /proc/self/cgroup 解析当前容器 ID
func SelfContainerID() string {
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
