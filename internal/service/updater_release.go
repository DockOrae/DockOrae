package service

import (
	"net/http"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/model"
)

// ---------- Release Notes 分类解析 ----------

// ParseReleaseNotes 将 GitHub Release Notes(markdown)解析为分类段:
// Features / Bug Fixes / Fixes / Improvements / Enhancements / Security / Breaking Changes。
// 分类解析失败(无任何分类命中)返回 nil,调用方回退显示原始 body——解析失败绝不阻塞更新。
func ParseReleaseNotes(body string) []model.ReleaseNoteSection {
	if strings.TrimSpace(body) == "" {
		return nil
	}
	var sections []model.ReleaseNoteSection
	var curType string
	var curItems []string
	flush := func() {
		if curType != "" && len(curItems) > 0 {
			// 同类型多段(如多个 Features 标题)合并为一个 section
			for i := range sections {
				if sections[i].Type == curType {
					sections[i].Items = append(sections[i].Items, curItems...)
					curType, curItems = "", nil
					return
				}
			}
			sections = append(sections, model.ReleaseNoteSection{Type: curType, Items: curItems})
		}
		curType, curItems = "", nil
	}
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if t == "" {
			continue
		}
		// 标题(# / ## / ###):切换分类
		if strings.HasPrefix(t, "#") {
			flush()
			curType = matchNoteType(strings.TrimLeft(t, "# "))
			continue
		}
		if curType == "" {
			continue // 标题前的正文不属于任何分类
		}
		if item := extractNoteItem(t); item != "" {
			curItems = append(curItems, item)
		}
	}
	flush()
	return sections
}

// matchNoteType 匹配标题关键词到分类 type(英文标题为主,兼容中文)
func matchNoteType(heading string) string {
	h := strings.ToLower(strings.TrimSpace(heading))
	switch {
	case containsAny(h, "feature", "new", "新增", "新功能", "功能"):
		return "features"
	case containsAny(h, "bug fix", "bugfix", "fix", "fixed", "修复", "问题修复"):
		return "bug_fixes"
	case containsAny(h, "improve", "enhance", "optimize", "optimization", "优化", "改进", "增强"):
		return "improvements"
	case containsAny(h, "security", "vulnerab", "安全", "漏洞"):
		return "security"
	case containsAny(h, "breaking", "incompat", "不兼容", "重要变更", "破坏"):
		return "breaking_changes"
	}
	return ""
}

var numberedItemRe = regexp.MustCompile(`^\d+[.)]\s*(.*)$`)

// extractNoteItem 提取列表项(- / * / + / 数字列表),非列表行返回空
func extractNoteItem(line string) string {
	t := strings.TrimSpace(line)
	if len(t) >= 2 && (t[0] == '-' || t[0] == '*' || t[0] == '+') && (t[1] == ' ' || t[1] == '\t') {
		return strings.TrimSpace(t[1:])
	}
	if m := numberedItemRe.FindStringSubmatch(t); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// ---------- 更新包可用性检查 ----------

// 二进制发布资产名(与 Makefile cross / release.yml 产物一致)
const binaryAssetPrefix = "docker-manager-go-linux-"

// binaryAssetName 当前平台对应的发布资产名(docker-manager-go-linux-<arch>.tar.gz)
func binaryAssetName() string {
	return binaryAssetPrefix + runtime.GOARCH + ".tar.gz"
}

// checksumAssetName 当前平台对应的 sha256 校验文件资产名
func checksumAssetName() string {
	return binaryAssetName() + ".sha256"
}

// findAsset 在 release assets 中按名查找
func findAsset(assets []model.ReleaseAsset, name string) *model.ReleaseAsset {
	for i := range assets {
		if assets[i].Name == name {
			return &assets[i]
		}
	}
	return nil
}

// defaultDockerRepo 默认镜像仓库(fallback:compose 文件未找到 image 时)
const defaultDockerRepo = "zhaoweiwen123/docker-manager-go"

// dockerImageAvailable 检查 Docker Hub 上是否存在指定 tag 的镜像(明确版本,不依赖 latest)
func dockerImageAvailable(tag string) bool {
	client := &http.Client{Timeout: 10 * time.Second}
	url := "https://hub.docker.com/v2/repositories/" + defaultDockerRepo + "/tags/" + strings.TrimPrefix(tag, "v")
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// installType 更新安装方式(API 展示用):binary / docker
func installType() string {
	if DeploymentMode() == "binary" {
		return "binary"
	}
	return "docker"
}

// checkInstallable 判断当前安装方式是否有可用的更新包:
//   - binary:release assets 中是否存在当前平台二进制
//   - docker:目标版本镜像是否已发布到 Docker Hub
func checkInstallable(assets []model.ReleaseAsset, tag string) (bool, string) {
	if installType() == "docker" {
		if dockerImageAvailable(tag) {
			return true, ""
		}
		return false, "Docker 镜像尚未发布"
	}
	if findAsset(assets, binaryAssetName()) != nil {
		return true, ""
	}
	return false, "当前平台暂无可用的更新包(" + runtime.GOARCH + ")"
}

var releaseTagRe = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

// validReleaseTag 校验 release tag 格式(防止恶意 tag 注入 shell 命令)
func validReleaseTag(tag string) bool {
	return releaseTagRe.MatchString(tag)
}
