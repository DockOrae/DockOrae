package service

import (
	"archive/tar"
	"bytes"
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
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/client"
	"golang.org/x/mod/semver"

	"github.com/DockOrae/DockOrae/internal/docker"
	"github.com/DockOrae/DockOrae/internal/model"
	"github.com/DockOrae/DockOrae/internal/state"
)

// AppVersion 面板当前版本,构建时由 ldflags 注入(与 main.Version 同步,发版打 tag 即可):
//
//	-X github.com/DockOrae/DockOrae/internal/service.AppVersion=v1.0.3
//
// 未注入(本地开发)时为空字符串,运行时按 unknown 处理。使用 Makefile 构建时自动注入。
var AppVersion string

// ---------- 更新状态(异步化 + 前端轮询) ----------

// UpdatePhase 更新进行阶段(前端按 phase 显示进度文案)
type UpdatePhase string

const (
	PhaseIdle        UpdatePhase = "idle"
	PhaseDownloading UpdatePhase = "downloading" // binary:下载 release 包
	PhaseVerifying   UpdatePhase = "verifying"   // binary:SHA256 校验
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
	Percent    int         `json:"percent"` // 0-100,前端进度条
	Error      string      `json:"error"`
	StartedAt  time.Time   `json:"started_at"`
	FinishedAt time.Time   `json:"finished_at"`
}

// phasePercent 各阶段的基准进度(下载阶段会按实际字节数实时推进到 40)
func phasePercent(p UpdatePhase) int {
	switch p {
	case PhaseDownloading:
		return 10
	case PhaseVerifying:
		return 45
	case PhaseExtracting:
		return 60
	case PhaseReplacing:
		return 80
	case PhaseRestarting:
		return 95
	case PhasePulling:
		return 15
	case PhaseHelper:
		return 70
	case PhaseDone:
		return 100
	}
	return 0
}

var (
	updateStatusMu sync.Mutex
	updateStatus   UpdateStatus
)

// setUpdatePhase 推进更新阶段(同时更新基准进度)
func setUpdatePhase(p UpdatePhase) {
	updateStatusMu.Lock()
	defer updateStatusMu.Unlock()
	updateStatus.Running = p != PhaseIdle && p != PhaseDone && p != PhaseFailed
	updateStatus.Phase = p
	updateStatus.Percent = phasePercent(p)
	updateStatus.Error = ""
	if p == PhaseDone || p == PhaseFailed {
		updateStatus.FinishedAt = time.Now()
	}
}

// updatePercent 仅推进进度(下载阶段实时调用,不改变阶段)
func updatePercent(pct int) {
	updateStatusMu.Lock()
	defer updateStatusMu.Unlock()
	if pct > updateStatus.Percent {
		updateStatus.Percent = pct
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
	updateGitHubURL  = "https://api.github.com/repos/DockOrae/DockOrae/releases/latest"
	updateCheckTTL   = 10 * time.Minute
	composeHelperImg = "docker/compose:latest"
	updateHelperName = "dm-update-helper"
	// maxUpdateExtractBytes 更新包解压体积上限(防御性,防压缩包炸弹)
	maxUpdateExtractBytes = 512 << 20
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

// CompareVersions 基于 SemVer 比较版本号("v1.0.3" / "1.0.2" / 空):
// a<b → -1,a==b → 0,a>b → 1。
// 正确处理 v1.10.0 > v1.2.0、v1.3.0 > v1.3.0-rc.1 > v1.3.0-alpha。
// 空/无效版本按 v0.0.0 处理(总是落后于任何正式版本)。
func CompareVersions(a, b string) int {
	return semver.Compare(normalizeSemVer(a), normalizeSemVer(b))
}

// normalizeSemVer 归一化为 golang.org/x/mod/semver 要求的 "vX.Y.Z" 格式
func normalizeSemVer(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if s == "" {
		return "v0.0.0"
	}
	v := "v" + s
	if semver.IsValid(v) {
		return v
	}
	return "v0.0.0" // 无效版本视为最老
}

// DisplayVersion 展示用版本号:构建未注入(本地开发/CI 检查)时为 unknown
func DisplayVersion() string {
	if AppVersion == "" {
		return "unknown"
	}
	return AppVersion
}

// UpdateCheck 检测 GitHub 最新 Release(结果缓存 10 分钟,防 GitHub API 限流;失败不缓存)
func UpdateCheck(st *state.AppState, ctx context.Context) (*model.UpdateInfo, error) {
	updateMu.Lock()
	defer updateMu.Unlock()
	if updateCache != nil && time.Since(updateCheckedAt) < updateCheckTTL {
		return updateCache, nil
	}

	info := &model.UpdateInfo{Current: DisplayVersion()}
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
				} else if rel.Draft {
					// draft 不对外发布,视为无更新
					info.Latest = DisplayVersion()
				} else if rel.Prerelease {
					// 预发布版本默认不提示普通用户(仅记录,不触发更新提示)
					info.Latest = DisplayVersion()
					info.Release = &rel
				} else {
					latest := rel.TagName
					info.Latest = latest
					info.Release = &rel
					info.InstallType = installType()
					info.HasUpdate = CompareVersions(AppVersion, latest) < 0
					// Release Notes 分类解析(失败回退原始 body,不影响更新检查)
					info.Notes = ParseReleaseNotes(rel.Body)
					info.NotesRaw = len(info.Notes) == 0 && strings.TrimSpace(rel.Body) != ""
					// 当前安装方式的更新包可用性:不可用则前端禁用"立即更新"
					if info.HasUpdate {
						info.Installable, info.NotInstallableReason = checkInstallable(st, rel.Assets, latest)
					} else {
						info.Installable = true
					}
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

	// 目标版本来自最近一次检查的缓存(带 v 的 tag,如 v1.0.3);未检查过则拒绝
	updateMu.Lock()
	tag := ""
	if updateCache != nil && updateCache.Release != nil {
		tag = updateCache.Release.TagName
	}
	updateMu.Unlock()
	if !validReleaseTag(tag) {
		updateStatusMu.Lock()
		updateStatus.Running = false
		updateStatus.Phase = PhaseFailed
		updateStatus.Error = "无法确定目标版本,请先检查更新"
		updateStatus.FinishedAt = time.Now()
		updateStatusMu.Unlock()
		return NewApiError(400, "无法确定目标版本,请先检查更新")
	}

	go func() {
		// 用独立 context:请求结束后 ctx 会被取消,后台任务不受影响
		log.Printf("update started: mode=%s version=%s current=%s", DeploymentMode(), tag, DisplayVersion())
		var err error
		if DeploymentMode() == "binary" {
			err = applyBinaryUpdate(st, context.Background(), tag)
		} else {
			err = applyComposeUpdate(st, context.Background(), tag)
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
			log.Printf("update failed: mode=%s version=%s error=%v", DeploymentMode(), tag, err)
		} else {
			log.Printf("update completed: mode=%s version=%s", DeploymentMode(), tag)
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

// ---------- Compose 更新:image tag 精确替换(修复 UPD-002) ----------

var imageLineRe = regexp.MustCompile(`^\s*image:\s*(\S+)\s*$`)

// findManagerImage 在 compose yaml 中查找 dockorae 的 image 值(第一个匹配)。
// 返回 image 字段的完整值(如 zhaoweiwen123/dockorae:latest);
// 找不到(未使用该镜像 / 值带引号等)返回 false。
// 只匹配镜像名(最后一段)为 dockorae 的条目,绝不误改 nginx/redis 等其他镜像。
func findManagerImage(yaml string) (string, bool) {
	for _, line := range strings.Split(yaml, "\n") {
		m := imageLineRe.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		val := m[1]
		// 去掉 :tag / @digest 取镜像名
		name := val
		if i := strings.IndexAny(name, "@:"); i >= 0 {
			name = name[:i]
		}
		if i := strings.LastIndex(name, "/"); i >= 0 {
			name = name[i+1:]
		}
		if name == "dockorae" {
			return val, true
		}
	}
	return "", false
}

// retagImageValue 把 image 值(可含 :tag 或 @digest)替换为指定 tag,保留 repository。
//
//	zhaoweiwen123/dockorae:latest      → zhaoweiwen123/dockorae:v1.3.0
//	registry.example.com/docker-manager-go:v1.0.2 → registry.example.com/docker-manager-go:v1.3.0
//	docker-manager-go@sha256:abc                 → docker-manager-go:v1.3.0
func retagImageValue(val, tag string) string {
	if i := strings.IndexAny(val, "@:"); i >= 0 {
		return val[:i] + ":" + tag
	}
	return val + ":" + tag
}

// safeImageValue 校验 image 值仅含安全字符(字母数字 . _ : @ / -),可安全嵌入 sed 模式。
func safeImageValue(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') ||
			r == '.' || r == '_' || r == ':' || r == '@' || r == '/' || r == '-' {
			continue
		}
		return false
	}
	return true
}

// buildHelperCmd 构造 helper 执行命令:备份 compose → 精确替换 docker-manager-go 的
// image 值(整行匹配,保留行内注释)→ compose up。
// oldVal/newVal 均已通过 safeImageValue 校验,可安全嵌入 sed。
func buildHelperCmd(oldVal, newVal string) string {
	return fmt.Sprintf(
		"cp /work/docker-compose.yml /work/docker-compose.yml.bak 2>/dev/null; "+
			"sed -i 's|^\\( *image: *\\)%s\\(.*\\)$|\\1%s\\2|' /work/docker-compose.yml; "+
			"docker compose -f /work/docker-compose.yml up -d --force-recreate --pull always",
		oldVal, newVal)
}

// readHostFile 读取宿主路径文件:容器内宿主根经 /host 只读挂载,binary 模式直接宿主路径。
func readHostFile(hostPath string) ([]byte, error) {
	if b, err := os.ReadFile("/host" + hostPath); err == nil {
		return b, nil
	}
	return os.ReadFile(hostPath)
}

// applyComposeUpdate compose 部署:独立 compose 容器代跑(面板容器重建期间执行者不受影响)。
// 更新目标为明确版本 tag(禁止依赖 latest):helper 先备份并替换 compose 中的镜像 tag,再 pull+up。
// 修复 UPD-001:等待 helper 退出并检查退出码,失败回传具体原因(不再"启动即成功")。
func applyComposeUpdate(st *state.AppState, ctx context.Context, tag string) error {
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

	// 3. 修复 UPD-002:读取宿主 compose 文件,精确识别 docker-manager-go 的 image 值并替换 tag。
	//    旧实现 sed 只替换 :latest,用户写死版本 tag(如 v1.0.2)时更新不生效(更新假成功)。
	yamlBytes, err := readHostFile(filepath.Join(dir, "docker-compose.yml"))
	if err != nil {
		return failUpdate("读取 docker-compose.yml 失败: %s", err.Error())
	}
	oldVal, ok := findManagerImage(string(yamlBytes))
	if !ok {
		return failUpdate("docker-compose.yml 中未找到 docker-manager-go 镜像(image 字段),已中止更新。请检查镜像名或手动修改 compose 后重试")
	}
	newVal := retagImageValue(oldVal, tag)
	if !safeImageValue(oldVal) || !safeImageValue(newVal) {
		return failUpdate("compose 镜像值包含非法字符,已中止更新: %q", oldVal)
	}
	helperCmd := buildHelperCmd(oldVal, newVal)

	// 4. 创建并启动 helper:挂载 docker.sock + compose 目录(可写)。
	//    面板容器对宿主 compose 目录只读(/:/host:ro),无法直接改文件,
	//    由 helper(root) 先备份 compose 文件,再替换 image tag,然后 up。
	//    不用 AutoRemove:失败后需要读取容器日志定位原因(见下方 WaitContainer 后取日志)。
	_, err = docker.CreateContainer(st.Docker, opCtx, client.ContainerCreateOptions{
		Config: &container.Config{
			Image:      composeHelperImg,
			Cmd:        []string{"sh", "-c", helperCmd},
			WorkingDir: "/work",
		},
		HostConfig: &container.HostConfig{
			Binds: []string{
				"/var/run/docker.sock:/var/run/docker.sock",
				dir + ":/work",
			},
		},
		Name: updateHelperName,
	})
	if err != nil {
		return failUpdate("创建更新容器失败: %s", err.Error())
	}
	if err := docker.StartContainer(st.Docker, opCtx, updateHelperName); err != nil {
		return failUpdate("启动更新容器失败: %s", err.Error())
	}

	// 4. 修复 UPD-001:等待 helper 真正执行完成(内部 pull 新镜像 + recreate 面板容器),
	//    检查退出码——启动成功 ≠ 更新成功(镜像不存在/拉取失败/权限不足等都会让 helper 失败)。
	waitCtx, waitCancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer waitCancel()
	code, waitErr := docker.WaitContainer(st.Docker, waitCtx, updateHelperName)
	if waitErr != nil {
		_ = docker.RemoveContainer(st.Docker, opCtx, updateHelperName, client.ContainerRemoveOptions{Force: true})
		return failUpdate("等待更新容器退出失败: %s", waitErr.Error())
	}
	if code != 0 {
		detail := helperLogsTail(st, updateHelperName, 80)
		_ = docker.RemoveContainer(st.Docker, opCtx, updateHelperName, client.ContainerRemoveOptions{Force: true})
		return failUpdate("Compose 更新失败(exit code %d): %s", code, detail)
	}
	// 成功:清理 helper 容器(保持环境干净)
	_ = docker.RemoveContainer(st.Docker, opCtx, updateHelperName, client.ContainerRemoveOptions{Force: true})

	// helper 已确认执行成功(pull + up 完成,面板容器已重建),标记完成供前端转轮询版本接口
	setUpdatePhase(PhaseDone)
	return nil
}

// helperLogsTail 读取 helper 容器最近的日志行(尽力而为;容器已删除时返回空串)。
// 供更新失败时定位原因(如 compose up 的具体报错)。
func helperLogsTail(st *state.AppState, id string, lines int) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	res, err := docker.ContainerLogs(st.Docker, ctx, id, client.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       strconv.Itoa(lines),
	})
	if err != nil {
		return ""
	}
	defer res.Close()
	var buf bytes.Buffer
	_, _ = stdcopy.StdCopy(&buf, &buf, res)
	out := strings.TrimSpace(buf.String())
	if out == "" {
		return "(无日志输出)"
	}
	// 只保留最后几行,避免把整段日志塞进错误信息
	all := strings.Split(out, "\n")
	if len(all) > lines {
		all = all[len(all)-lines:]
	}
	return strings.Join(all, "\n")
}

// applyBinaryUpdate binary(systemd)部署:下载 GitHub Release 二进制资产,
// 校验后原子替换自身(/proc/self/exe),1.5 秒后 systemctl restart 服务。
// 阶段进度经 setUpdatePhase 上报,错误经 failUpdate 记录。
// 修复 UPD-003:替换后注册 systemd 一次性健康检查单元(独立于本进程),
// 面板重启后自动验证新版本;失败自动回滚 exe.old 并重启服务。
func applyBinaryUpdate(st *state.AppState, ctx context.Context, tag string) error {
	arch := runtime.GOARCH
	if arch != "amd64" && arch != "arm64" {
		return failUpdate("不支持的架构: %s", arch)
	}
	pkg := fmt.Sprintf("docker-manager-go-linux-%s.tar.gz", arch)
	// 修复 UPD-005:下载"已确认的版本"资产(tag 精确),不用 latest——
	// 避免检查后 GitHub 发布新版本导致下载错版本
	url := fmt.Sprintf("https://github.com/DockOrae/DockOrae/releases/download/%s/%s", tag, pkg)

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
	// 下载进度:响应头给出总大小时按字节实时推进 percent(10→40)
	if resp.ContentLength > 0 {
		_, err = io.Copy(tmpFile, io.TeeReader(resp.Body, &progressWriter{total: resp.ContentLength, from: 10, to: 40}))
	} else {
		_, err = io.Copy(tmpFile, resp.Body)
	}
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return failUpdate("下载中断: %s", err.Error())
	}
	tmpFile.Close()

	// 2. SHA256 校验(校验失败立即中止,绝不替换当前运行文件)
	setUpdatePhase(PhaseVerifying)
	if err := verifySHA256(tmpFile.Name(), url+".sha256", client); err != nil {
		os.Remove(tmpFile.Name())
		return failUpdate("%s", err.Error())
	}

	// 3. 解压出新的可执行文件
	setUpdatePhase(PhaseExtracting)
	newBin, err := extractBinary(tmpFile.Name())
	os.Remove(tmpFile.Name())
	if err != nil {
		return failUpdate("解压更新包失败: %s", err.Error())
	}
	defer os.Remove(newBin)

	// 4. 定位当前二进制并原子替换(先备份,失败可回滚)
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

	// 5. 延迟重启服务 + 外部健康检查(UPD-003):
	//    systemd-run 注册一次性检查单元(独立进程,不依赖本进程存活),
	//    面板重启后验证新版本健康;失败自动回滚 exe.old 并重启服务。
	setUpdatePhase(PhaseRestarting)
	scheduleBinaryHealthCheck(st, exe, exe+".old", tag)
	go func() {
		time.Sleep(1500 * time.Millisecond)
		if err := exec.Command("systemctl", "restart", "docker-manager").Run(); err != nil {
			log.Printf("update: systemctl restart docker-manager 失败: %v (可手动执行恢复)", err)
		}
	}()
	return nil
}

// buildHealthCheckScript 构造 systemd 一次性检查单元的脚本:
// 面板重启后 sleep 8s 等服务起来 → curl /api/health 取 version →
//
//	版本 == 期望值(且非 unknown)→ 健康,正常退出;
//	否则(服务未起/版本不对)→ 回滚 exe.old → 重启服务 → 退出 1。
//
// port/expect/old/exe 均为内部值(expect 经 validReleaseTag 校验),嵌入安全。
func buildHealthCheckScript(port int, expect, old, exe string) string {
	return fmt.Sprintf(
		`sleep 8; ver=$(curl -fsSL --max-time 5 http://127.0.0.1:%d/api/health 2>/dev/null | grep -o '"version":"[^"]*"' | head -1 | cut -d'"' -f4); `+
			`if [ -n "$ver" ] && [ "$ver" != "unknown" ] && [ "$ver" = "%s" ]; then `+
			`echo "update: new version %s healthy"; exit 0; fi; `+
			`echo "update: health check failed (version=$ver), rolling back to %s"; `+
			`cp -f %s %s; chmod +x %s; systemctl restart docker-manager; exit 1`,
		port, expect, expect, old, old, exe, exe)
}

// scheduleBinaryHealthCheck 注册 systemd 一次性健康检查(UPD-003):
// 面板进程替换二进制后即将重启,重启后的验证必须由独立机制完成(本进程已不在)。
// 用 systemd-run --on-active 启动延迟的一次性单元,失败自动回滚。
// 非 systemd 环境(systemd-run 缺失)降级:仅告警,exe.old 保留供手动恢复。
func scheduleBinaryHealthCheck(st *state.AppState, exe, old, expect string) {
	if _, err := exec.LookPath("systemd-run"); err != nil {
		log.Printf("update: systemd-run 不可用,跳过自动健康检查/回滚(旧二进制保留于 %s,失败时可手动恢复)", old)
		return
	}
	script := buildHealthCheckScript(int(st.Settings.Get().WebPort), expect, old, exe)
	unit := fmt.Sprintf("dm-update-check-%d", time.Now().UnixNano()%1000000)
	cmd := exec.Command("systemd-run", "--on-active=10", "--unit="+unit, "--collect", "/bin/sh", "-c", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("update: 注册 systemd 健康检查失败(可手动恢复 %s): %v %s", old, err, strings.TrimSpace(string(out)))
	} else {
		log.Printf("update: 已注册 systemd 健康检查单元 %s(重启后验证新版本,失败自动回滚 %s)", unit, old)
	}
}

// progressWriter 下载进度统计:每 256KB 更新一次状态 percent(节流,避免频繁加锁)
type progressWriter struct {
	total    int64
	done     int64
	from, to int
	next     int64
}

func (w *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	w.done += int64(n)
	if w.total > 0 && w.done >= w.next {
		w.next = w.done + 256*1024
		pct := w.from + int(float64(w.done)/float64(w.total)*float64(w.to-w.from))
		if pct > w.to {
			pct = w.to
		}
		updatePercent(pct)
	}
	return n, nil
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
	var total int64
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
		// UPD-007:限制解压体积,防压缩包炸弹(源为官方 release,防御性限制)
		total += hdr.Size
		if total > maxUpdateExtractBytes {
			return "", fmt.Errorf("更新包解压体积超过限制(%d bytes)", maxUpdateExtractBytes)
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
