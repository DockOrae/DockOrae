package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DockerManger/Docker_Manager_Go/internal/model"
)

// ---------- CompareVersions(SemVer) ----------

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		// 空/无效版本按 v0.0.0 处理
		{"", "v1.0.0", -1},
		{"", "", 0},
		{"v1.0.0", "", 1},
		// 基础比较
		{"v1.0.9", "v1.0.10", -1}, // 数字位比较,非字典序
		{"1.0.10", "v1.0.9", 1},   // 无 v 前缀兼容
		{"v1.2.0", "v1.10.0", -1},
		{"v2.0.0", "v1.99.99", 1},
		{"v1.0.0", "v1.0.0", 0},
		// 预发布排序:稳定 > rc > beta > alpha
		{"v1.3.0-rc.1", "v1.3.0", -1},
		{"v1.3.0-alpha", "v1.3.0-rc.1", -1},
		{"v1.3.0-beta.2", "v1.3.0-rc.1", -1},
		{"v1.3.0-rc.1", "v1.3.0-rc.2", -1},
		// 构建元数据不影响比较
		{"v1.0.0+abc", "v1.0.0", 0},
	}
	for _, c := range cases {
		got := CompareVersions(c.a, c.b)
		if got != c.want {
			t.Errorf("CompareVersions(%q, %q) = %d, want %d", c.a, c.b, got, c.want)
		}
		// 对称性
		if got != 0 && CompareVersions(c.b, c.a) != -got {
			t.Errorf("CompareVersions(%q, %q) 不对称", c.b, c.a)
		}
	}
}

func TestNormalizeSemVer(t *testing.T) {
	cases := map[string]string{
		"":            "v0.0.0",
		"v1.2.3":      "v1.2.3",
		"1.2.3":       "v1.2.3",
		"  v1.2.3  ":  "v1.2.3",
		"v1.2.3-rc.1": "v1.2.3-rc.1",
		"not-a-ver":   "v0.0.0", // 无效 → 最老
		"dev":         "v0.0.0",
	}
	for in, want := range cases {
		if got := normalizeSemVer(in); got != want {
			t.Errorf("normalizeSemVer(%q) = %q, want %q", in, got, want)
		}
	}
}

// ---------- Release Notes 分类解析 ----------

func TestParseReleaseNotes(t *testing.T) {
	body := `## Features

- Add online update
- Add Docker Compose update

## Bug Fixes

- Fix container status detection

## Improvements

- Improve Docker API performance

## Security

- Fix authentication issue

## Breaking Changes

- Requires Go 1.27

## 优化

- 中文标题也支持

### 修复

- 三级标题修复项
`
	sections := ParseReleaseNotes(body)
	if len(sections) != 5 { // features/bug_fixes/improvements/security/breaking_changes(中文"优化"合并进 improvements,"修复"合并进 bug_fixes)
		t.Fatalf("sections = %d, want 5", len(sections))
	}
	byType := map[string][]string{}
	for _, s := range sections {
		byType[s.Type] = s.Items
	}
	if len(byType["features"]) != 2 {
		t.Errorf("features items = %v, want 2", byType["features"])
	}
	if len(byType["bug_fixes"]) != 2 { // 英文 Fix + 中文"修复"(三级标题)合并
		t.Errorf("bug_fixes items = %v, want 2", byType["bug_fixes"])
	}
	if len(byType["improvements"]) != 2 { // 英文 Improvements + 中文"优化"合并
		t.Errorf("improvements items = %v, want 2", byType["improvements"])
	}
	if len(byType["security"]) != 1 {
		t.Errorf("security items = %v, want 1", byType["security"])
	}
	if len(byType["breaking_changes"]) != 1 {
		t.Errorf("breaking_changes items = %v, want 1", byType["breaking_changes"])
	}
}

func TestParseReleaseNotesFallback(t *testing.T) {
	// 空 body → nil(前端回退原始 body)
	if got := ParseReleaseNotes(""); got != nil {
		t.Errorf("empty body = %v, want nil", got)
	}
	if got := ParseReleaseNotes("   \n  "); got != nil {
		t.Errorf("blank body = %v, want nil", got)
	}
	// 无分类标题的纯文本 → nil(回退原始 body,不阻塞更新检查)
	plain := "本次更新修复了一些问题。\n- 一些优化"
	if got := ParseReleaseNotes(plain); got != nil {
		t.Errorf("plain body = %v, want nil", got)
	}
	// 有分类标题但无列表项 → nil(无内容可展示)
	headOnly := "## Features\n\n## Bug Fixes\n"
	if got := ParseReleaseNotes(headOnly); got != nil {
		t.Errorf("head-only body = %v, want nil", got)
	}
	// 长 body(多段+空行)不 panic
	long := "## Features\n" + strings.Repeat("- item\n", 500) + "\n## Security\n- x\n"
	if got := ParseReleaseNotes(long); len(got) != 2 {
		t.Errorf("long body sections = %v, want 2", len(got))
	}
}

func TestValidReleaseTag(t *testing.T) {
	valid := []string{"v1.0.3", "1.0.3", "v1.2.3-rc.1", "v1.0.3+build.5"}
	invalid := []string{"", "latest", "v1.0", "v1.0.3;rm -rf /", "v1.0.3|evil", "v1.0.3 $(id)", "..", "v"}
	for _, tag := range valid {
		if !validReleaseTag(tag) {
			t.Errorf("validReleaseTag(%q) = false, want true", tag)
		}
	}
	for _, tag := range invalid {
		if validReleaseTag(tag) {
			t.Errorf("validReleaseTag(%q) = true, want false", tag)
		}
	}
}

// ---------- SHA256 校验 ----------

func TestVerifySHA256(t *testing.T) {
	content := []byte("docker-manager-go binary content")
	sum := sha256.Sum256(content)
	goodSum := hex.EncodeToString(sum[:])

	dir := t.TempDir()
	file := filepath.Join(dir, "pkg.tar.gz")
	if err := os.WriteFile(file, content, 0o644); err != nil {
		t.Fatal(err)
	}

	// 1. 校验一致 → 通过
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  docker-manager-go-linux-amd64.tar.gz\n", goodSum)
	}))
	defer srv.Close()
	if err := verifySHA256(file, srv.URL, srv.Client()); err != nil {
		t.Errorf("good checksum failed: %v", err)
	}

	// 2. 校验不一致 → ErrChecksumMismatch
	srvBad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  x\n", strings.Repeat("0", 64))
	}))
	defer srvBad.Close()
	err := verifySHA256(file, srvBad.URL, srvBad.Client())
	if !errors.Is(err, ErrChecksumMismatch) {
		t.Errorf("bad checksum err = %v, want ErrChecksumMismatch", err)
	}

	// 3. 校验文件 404 → 跳过(不报错,向后兼容无校验资产的老 release)
	srv404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv404.Close()
	if err := verifySHA256(file, srv404.URL, srv404.Client()); err != nil {
		t.Errorf("404 checksum should skip, got %v", err)
	}
}

// ---------- 资产匹配 ----------

func TestAssetMatching(t *testing.T) {
	assets := []model.ReleaseAsset{
		{Name: "docker-manager-go-linux-amd64.tar.gz"},
		{Name: "docker-manager-go-linux-amd64.tar.gz.sha256"},
		{Name: "docker-manager-go-linux-arm64.tar.gz"},
	}
	if findAsset(assets, "docker-manager-go-linux-amd64.tar.gz") == nil {
		t.Error("amd64 asset not found")
	}
	if findAsset(assets, "docker-manager-go-linux-s390x.tar.gz") != nil {
		t.Error("s390x asset should not exist")
	}
	if findAsset(nil, "x") != nil {
		t.Error("nil assets should return nil")
	}
}

func TestChecksumAssetName(t *testing.T) {
	got := checksumAssetName()
	if !strings.HasSuffix(got, ".tar.gz.sha256") {
		t.Errorf("checksumAssetName() = %q, want *.tar.gz.sha256", got)
	}
}

// ---------- Compose image tag 精确替换(UPD-002) ----------

func TestFindManagerImage(t *testing.T) {
	yaml := `services:
  nginx:
    image: nginx:latest
  redis:
    image: redis:7-alpine
  docker-manager:
    image: zhaoweiwen123/docker-manager-go:latest
  db:
    image: postgres:16
`
	got, ok := findManagerImage(yaml)
	if !ok || got != "zhaoweiwen123/docker-manager-go:latest" {
		t.Fatalf("findManagerImage = %q, %v; want zhaoweiwen123/docker-manager-go:latest, true", got, ok)
	}

	// 写死版本 tag 也能识别
	yaml2 := strings.ReplaceAll(yaml, "zhaoweiwen123/docker-manager-go:latest", "zhaoweiwen123/docker-manager-go:v1.0.2")
	if got, ok = findManagerImage(yaml2); !ok || got != "zhaoweiwen123/docker-manager-go:v1.0.2" {
		t.Fatalf("pinned tag: findManagerImage = %q, %v", got, ok)
	}

	// 自定义 registry + 无 tag
	yaml3 := `services:
  app:
    image: registry.example.com/team/docker-manager-go
`
	if got, ok = findManagerImage(yaml3); !ok || got != "registry.example.com/team/docker-manager-go" {
		t.Fatalf("custom registry: findManagerImage = %q, %v", got, ok)
	}

	// digest 形式
	yaml4 := `services:
  app:
    image: zhaoweiwen123/docker-manager-go@sha256:abcdef1234567890
`
	if got, ok = findManagerImage(yaml4); !ok || got != "zhaoweiwen123/docker-manager-go@sha256:abcdef1234567890" {
		t.Fatalf("digest: findManagerImage = %q, %v", got, ok)
	}

	// 没有 docker-manager-go → false
	noManager := `services:
  web:
    image: nginx:latest
  cache:
    image: redis:7-alpine
`
	if got, ok := findManagerImage(noManager); ok {
		t.Fatalf("no manager image: got %q, want false", got)
	}

	// 空 yaml / 无 image 字段
	if _, ok := findManagerImage("services:\n  web:\n    build: .\n"); ok {
		t.Fatal("build-only compose should not match")
	}
}

func TestRetagImageValue(t *testing.T) {
	cases := []struct{ val, tag, want string }{
		{"zhaoweiwen123/docker-manager-go:latest", "v1.3.0", "zhaoweiwen123/docker-manager-go:v1.3.0"},
		{"zhaoweiwen123/docker-manager-go:v1.0.2", "v1.3.0", "zhaoweiwen123/docker-manager-go:v1.3.0"},
		{"zhaoweiwen123/docker-manager-go:v1.2.3", "v1.3.0", "zhaoweiwen123/docker-manager-go:v1.3.0"},
		{"zhaoweiwen123/docker-manager-go:v1.10.20", "v1.3.0", "zhaoweiwen123/docker-manager-go:v1.3.0"},
		{"registry.example.com/docker-manager-go:v1.0.2", "v1.3.0", "registry.example.com/docker-manager-go:v1.3.0"},
		{"docker-manager-go", "v1.3.0", "docker-manager-go:v1.3.0"},
		{"zhaoweiwen123/docker-manager-go@sha256:abc", "v1.3.0", "zhaoweiwen123/docker-manager-go:v1.3.0"},
	}
	for _, c := range cases {
		if got := retagImageValue(c.val, c.tag); got != c.want {
			t.Errorf("retagImageValue(%q, %q) = %q, want %q", c.val, c.tag, got, c.want)
		}
	}
}

func TestSafeImageValue(t *testing.T) {
	valid := []string{"docker-manager-go:latest", "zhaoweiwen123/docker-manager-go:v1.0.2", "registry.example.com/a/docker-manager-go@sha256:abc", "docker-manager-go"}
	for _, v := range valid {
		if !safeImageValue(v) {
			t.Errorf("safeImageValue(%q) = false, want true", v)
		}
	}
	invalid := []string{"", "nginx:latest;rm -rf /", "a b", "a\"b", "a$(id)", "a`id`"}
	for _, v := range invalid {
		if safeImageValue(v) {
			t.Errorf("safeImageValue(%q) = true, want false", v)
		}
	}
}

func TestBuildHelperCmd(t *testing.T) {
	cmd := buildHelperCmd("zhaoweiwen123/docker-manager-go:latest", "zhaoweiwen123/docker-manager-go:v1.3.0")
	// 包含精确替换(整行 image: 值)而不是脆弱的 :latest 全局替换
	if !strings.Contains(cmd, "zhaoweiwen123/docker-manager-go:latest") {
		t.Errorf("helper cmd 缺少旧值: %s", cmd)
	}
	if !strings.Contains(cmd, "zhaoweiwen123/docker-manager-go:v1.3.0") {
		t.Errorf("helper cmd 缺少新值: %s", cmd)
	}
	// 新值只出现在替换目标中(不会再次匹配 image 行导致二次替换)
	if strings.Count(cmd, "docker-manager-go:v1.3.0") != 1 {
		t.Errorf("新值出现次数异常: %s", cmd)
	}
	if !strings.Contains(cmd, "docker compose -f /work/docker-compose.yml up -d --force-recreate --pull always") {
		t.Errorf("helper cmd 缺少 compose up: %s", cmd)
	}
}

// ---------- Binary 更新健康检查脚本(UPD-003) ----------

func TestBuildHealthCheckScript(t *testing.T) {
	script := buildHealthCheckScript(8080, "v1.3.0", "/usr/local/bin/docker-manager-go.old", "/usr/local/bin/docker-manager-go")
	// 健康检查:curl health + 版本比较
	if !strings.Contains(script, "curl -fsSL --max-time 5 http://127.0.0.1:8080/api/health") {
		t.Errorf("缺少 health curl: %s", script)
	}
	if !strings.Contains(script, `"$ver" = "v1.3.0"`) {
		t.Errorf("缺少版本比较: %s", script)
	}
	if !strings.Contains(script, `[ "$ver" != "unknown" ]`) {
		t.Errorf("缺少 unknown 防护: %s", script)
	}
	// 回滚:cp 旧二进制回原位 + 重启
	if !strings.Contains(script, "cp -f /usr/local/bin/docker-manager-go.old /usr/local/bin/docker-manager-go") {
		t.Errorf("缺少回滚 cp: %s", script)
	}
	if !strings.Contains(script, "systemctl restart docker-manager") {
		t.Errorf("缺少重启: %s", script)
	}
	// shell 变量引用正确(没有残留 Go 占位符)
	if strings.Contains(script, "%s") || strings.Contains(script, "%d") {
		t.Errorf("脚本残留格式占位符: %s", script)
	}
	// 期望版本含特殊字符时(validReleaseTag 已限制)不破坏脚本结构
	script2 := buildHealthCheckScript(8080, "v1.3.0-rc.1", "/opt/dm/dm-go.old", "/opt/dm/dm-go")
	if !strings.Contains(script2, `"$ver" = "v1.3.0-rc.1"`) {
		t.Errorf("rc 版本比较缺失: %s", script2)
	}
}
