package api

import (
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/appstore"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
)

// appstoreList 应用列表 + 分类
func appstoreList(c *gin.Context, d *Deps) error {
	items, cats, err := d.AppStore.List()
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"apps": items, "categories": cats})
	return nil
}

// appstoreSync 同步应用商店数据(从 GitHub 仓库拉取)
func appstoreSync(c *gin.Context, d *Deps) error {
	if err := d.AppStore.Sync(); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// appstoreIcon 应用图标(公开接口,供 <img> 直接引用;失败返回 404,前端回退 emoji)
func appstoreIcon(c *gin.Context, d *Deps) error {
	b, err := appstore.FetchIcon(c.Param("key"), filepath.Join(d.St.Cfg.DataDir, "appstore"), d.St.Cfg.DataDir)
	if err != nil {
		return service.NewApiError(404, "appStore.notFound")
	}
	c.Data(200, "image/png", b)
	return nil
}

// appstoreDetail 应用详情(参数 schema)
func appstoreDetail(c *gin.Context, d *Deps) error {
	info, err := d.AppStore.Detail(c.Param("key"))
	if err != nil {
		return err
	}
	c.JSON(200, info)
	return nil
}

// appstoreInstall 一键安装
func appstoreInstall(c *gin.Context, d *Deps) error {
	var req struct {
		Params map[string]string `json:"params"`
		Yaml   string            `json:"yaml"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := d.AppStore.Install(c.Request.Context(), c.Param("key"), req.Params, req.Yaml); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// appstorePreview 渲染 compose 预览
func appstorePreview(c *gin.Context, d *Deps) error {
	var req struct {
		Params map[string]string `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	yaml, err := d.AppStore.Preview(c.Param("key"), req.Params)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"yaml": yaml})
	return nil
}

// appstoreUninstall 卸载
func appstoreUninstall(c *gin.Context, d *Deps) error {
	if err := d.AppStore.Uninstall(c.Param("key")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// appstoreUpgrade 升级
func appstoreUpgrade(c *gin.Context, d *Deps) error {
	if err := d.AppStore.Upgrade(c.Request.Context(), c.Param("key")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}
