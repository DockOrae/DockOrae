package service

import (
	"context"
	"os"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/appstore"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// AppStoreList 应用列表(附安装状态)
func AppStoreList(st *state.AppState) ([]map[string]any, error) {
	installed := installedApps(st)
	out := make([]map[string]any, 0, len(appstore.All()))
	for _, a := range appstore.All() {
		out = append(out, map[string]any{
			"key":         a.Key,
			"name":        a.Name,
			"icon":        a.Icon,
			"category":    a.Category,
			"description": a.Description,
			"ports":       a.Ports,
			"installed":   installed[a.Key],
		})
	}
	return out, nil
}

// AppStoreDetail 应用详情(含参数 schema)
func AppStoreDetail(key string) (map[string]any, error) {
	a := appstore.Get(key)
	if a == nil {
		return nil, BadRequest("appstore.notFound")
	}
	return map[string]any{
		"key":         a.Key,
		"name":        a.Name,
		"icon":        a.Icon,
		"category":    a.Category,
		"description": a.Description,
		"ports":       a.Ports,
		"params":      a.Params,
	}, nil
}

// AppStoreInstall 一键安装:渲染 compose 模板 → 保存到面板目录 → compose up。
// 受许可证限制(与创建容器/Compose 部署一致)
func AppStoreInstall(st *state.AppState, ctx context.Context, key string, params map[string]string) error {
	if !LicenseActive(st) {
		return NewApiError(403, "license.required")
	}
	a := appstore.Get(key)
	if a == nil {
		return BadRequest("appstore.notFound")
	}
	yaml, err := appstore.Render(a, params)
	if err != nil {
		return err
	}
	if err := ComposeAdopt(st, key, yaml); err != nil {
		return err
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

// AppStoreUpgrade 升级:重新拉取镜像并重建容器
func AppStoreUpgrade(st *state.AppState, key string) error {
	_, err := RunCompose(st, key, "up", "-d", "--force-recreate", "--pull", "always")
	return err
}

// installedApps 已安装应用集合(面板数据目录下的应用项目)
func installedApps(st *state.AppState) map[string]bool {
	out := map[string]bool{}
	entries, err := os.ReadDir(ProjectDir(st, ""))
	if err != nil {
		return out
	}
	for _, e := range entries {
		if e.IsDir() && appstore.Get(e.Name()) != nil {
			out[e.Name()] = true
		}
	}
	return out
}
