package service

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/appstore"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/docker"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
	"github.com/moby/moby/client"
)

// AppStoreDir 应用商店数据目录(仓库根,内含 apps/)
func AppStoreDir(st *state.AppState) string {
	return filepath.Join(st.Cfg.DataDir, "appstore")
}

// AppStoreSync 同步应用商店数据(从 GitHub 仓库拉取 tarball 解压)
func AppStoreSync(st *state.AppState) error {
	return appstore.Sync(AppStoreDir(st))
}

// AppStoreList 应用列表(附安装状态)+ 分类
func AppStoreList(st *state.AppState) ([]map[string]any, []string, error) {
	apps, cats, err := appstore.LoadApps(AppStoreDir(st))
	if err != nil {
		return nil, nil, err
	}
	installed := installedApps(st)
	out := make([]map[string]any, 0, len(apps))
	for _, a := range apps {
		out = append(out, map[string]any{
			"key":              a.Key,
			"name":             a.Name,
			"icon":             a.Icon,
			"category":         a.Category,
			"description":      a.Description,
			"ports":            a.Ports,
			"versions":         a.Versions,
			"installed":        installed[a.Key],
			"update_available": updateAvailable(st, a, installed[a.Key]),
		})
	}
	return out, cats, nil
}

// AppStoreDetail 应用详情(参数 schema + 版本列表)
func AppStoreDetail(st *state.AppState, key string) (map[string]any, error) {
	a, err := appstore.LoadApp(AppStoreDir(st), key)
	if err != nil {
		return nil, BadRequest("appstore.notFound")
	}
	params := []appstore.Param{}
	if len(a.Versions) > 0 {
		opts := make([]appstore.Option, 0, len(a.Versions))
		for _, v := range a.Versions {
			opts = append(opts, appstore.Option{Label: v, Value: v})
		}
		params = append(params, appstore.Param{Key: "version", LabelZh: "版本", LabelEn: "Version", Type: "select", Default: a.Versions[0], Options: opts})
	}
	params = append(params, a.Params...)
	installed := installedApps(st)[key]
	return map[string]any{
		"key":              a.Key,
		"name":             a.Name,
		"icon":             a.Icon,
		"category":         a.Category,
		"description":      a.Description,
		"ports":            a.Ports,
		"versions":         a.Versions,
		"installed":        installed,
		"update_available": updateAvailable(st, a, installed),
		"params":           append(params, appstore.GlobalParams...),
	}, nil
}

// AppStorePreview 渲染 compose 预览(不落盘、不部署)
func AppStorePreview(st *state.AppState, key string, params map[string]string) (string, error) {
	a, err := appstore.LoadApp(AppStoreDir(st), key)
	if err != nil {
		return "", BadRequest("appstore.notFound")
	}
	return appstore.Render(a, versionOf(a, params), params)
}

// AppStoreInstall 一键安装:渲染 compose → 保存到面板目录 → 复制附加文件 → compose up。
func AppStoreInstall(st *state.AppState, ctx context.Context, key string, params map[string]string, yamlOverride string) error {
	if !LicenseActive(st) {
		return NewApiError(403, "license.required")
	}
	a, err := appstore.LoadApp(AppStoreDir(st), key)
	if err != nil {
		return BadRequest("appstore.notFound")
	}
	version := versionOf(a, params)
	vd := a.VersionData(version)
	if vd == nil {
		return BadRequest("appstore.versionNotFound")
	}
	if err := appstore.ValidateParams(vd.Params, params); err != nil {
		if pe, ok := err.(*appstore.ParamError); ok {
			return NewApiError(400, "appstore.param."+pe.Msg)
		}
		return err
	}
	yaml, err := appstore.Render(a, version, params)
	if err != nil {
		return err
	}
	if strings.TrimSpace(yamlOverride) != "" {
		yaml = yamlOverride
	}
	if err := ComposeAdopt(st, key, yaml); err != nil {
		return err
	}
	// 复制版本目录的附加文件(conf/scripts 等)到项目目录
	if err := copyVersionFiles(st, key, vd.Dir); err != nil {
		return err
	}
	params["version"] = version
	if err := saveAppParams(st, key, params); err != nil {
		return err
	}
	// 确保 compose 引用的 external 网络存在(1Panel 同款 1panel-network)
	if err := ensureExternalNetworks(st, ctx, yaml); err != nil {
		return err
	}
	// 启动前拉取镜像(默认开启,未显式关闭则执行)
	if params["pull_first"] != "false" {
		_, _ = RunCompose(st, key, "pull")
	}
	if _, err := RunCompose(st, key, "up", "-d"); err != nil {
		return err
	}
	return nil
}

// AppStoreUninstall 卸载(停栈并删除编排文件)
func AppStoreUninstall(st *state.AppState, key string) error {
	_, _ = RunCompose(st, key, "down")
	return os.RemoveAll(ProjectDir(st, key))
}

// AppStoreUpgrade 升级:已安装版本非最新时重渲染最新版 compose + 复制附加文件;始终拉取镜像并重建容器
func AppStoreUpgrade(st *state.AppState, ctx context.Context, key string) error {
	a, err := appstore.LoadApp(AppStoreDir(st), key)
	if err != nil {
		return BadRequest("appstore.notFound")
	}
	params := loadAppParams(st, key)
	if len(a.Versions) > 1 && params["version"] != a.Versions[0] {
		params["version"] = a.Versions[0]
		yaml, err := appstore.Render(a, a.Versions[0], params)
		if err != nil {
			return err
		}
		if err := os.WriteFile(ComposeFile(st, key), []byte(yaml), 0o644); err != nil {
			return err
		}
		if vd := a.VersionData(a.Versions[0]); vd != nil {
			if err := copyVersionFiles(st, key, vd.Dir); err != nil {
				return err
			}
		}
		if err := saveAppParams(st, key, params); err != nil {
			return err
		}
	}
	if _, err := RunCompose(st, key, "up", "-d", "--force-recreate", "--pull", "always"); err != nil {
		return err
	}
	return nil
}

// ---------- 内部工具 ----------

func versionOf(a *appstore.App, params map[string]string) string {
	if v := strings.TrimSpace(params["version"]); v != "" {
		return v
	}
	if len(a.Versions) > 0 {
		return a.Versions[0]
	}
	return ""
}

// copyVersionFiles 把版本目录下的附加文件(conf/scripts 等)复制到项目目录,
// 跳过 compose/data.yml/README/logo(compose 相对挂载 ./conf ./data 依赖这些文件)
func copyVersionFiles(st *state.AppState, key, versionDir string) error {
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return nil
	}
	dest := ProjectDir(st, key)
	for _, e := range entries {
		name := e.Name()
		if name == "docker-compose.yml" || name == "data.yml" || strings.HasPrefix(name, "README") || name == "logo.png" {
			continue
		}
		if err := copyRecursive(filepath.Join(versionDir, name), filepath.Join(dest, name)); err != nil {
			return err
		}
	}
	return nil
}

func copyRecursive(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return nil
	}
	if info.IsDir() {
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dst, 0o755); err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyRecursive(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, info.Mode().Perm())
}

var externalNetRe = regexp.MustCompile(`(?m)^\s{2}([a-zA-Z0-9_.-]+):\s*\n\s{4}external:\s*true`)

// ensureExternalNetworks 确保 compose 引用的 external 网络存在(不存在则创建)
func ensureExternalNetworks(st *state.AppState, ctx context.Context, yaml string) error {
	seen := map[string]bool{}
	for _, m := range externalNetRe.FindAllStringSubmatch(yaml, -1) {
		name := strings.TrimSpace(m[1])
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		_, _, err := docker.InspectNetwork(st.Docker, ctx, name)
		if err == nil {
			continue
		}
		_, err = docker.CreateNetwork(st.Docker, ctx, name, client.NetworkCreateOptions{})
		if err != nil {
			return err
		}
	}
	return nil
}

// updateAvailable 已安装且存在比当前更新的版本
func updateAvailable(st *state.AppState, a *appstore.App, installed bool) bool {
	if !installed || len(a.Versions) < 2 {
		return false
	}
	return loadAppParams(st, a.Key)["version"] != a.Versions[0]
}

func loadAppParams(st *state.AppState, key string) map[string]string {
	params := map[string]string{}
	if b, err := os.ReadFile(filepath.Join(ProjectDir(st, key), "params.json")); err == nil {
		_ = json.Unmarshal(b, &params)
	}
	return params
}

func saveAppParams(st *state.AppState, key string, params map[string]string) error {
	b, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(ProjectDir(st, key), "params.json"), b, 0o644)
}

// installedApps 已安装应用集合(面板数据目录下的应用项目)
func installedApps(st *state.AppState) map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir(ProjectDir(st, ""))
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() {
			out[e.Name()] = true
		}
	}
	return out
}
