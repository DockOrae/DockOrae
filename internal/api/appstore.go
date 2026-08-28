package api

import (
	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/appstore"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// appstoreList 应用列表 + 分类
func appstoreList(c *gin.Context, st *state.AppState) error {
	items, err := service.AppStoreList(st)
	if err != nil {
		return err
	}
	c.JSON(200, gin.H{"apps": items, "categories": appstore.Categories()})
	return nil
}

// appstoreDetail 应用详情(参数 schema)
func appstoreDetail(c *gin.Context, st *state.AppState) error {
	info, err := service.AppStoreDetail(c.Param("key"))
	if err != nil {
		return err
	}
	c.JSON(200, info)
	return nil
}

// appstoreInstall 一键安装
func appstoreInstall(c *gin.Context, st *state.AppState) error {
	var req struct {
		Params map[string]string `json:"params"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		return service.BadRequest("err.requestFailed")
	}
	if err := service.AppStoreInstall(st, c.Request.Context(), c.Param("key"), req.Params); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// appstoreUninstall 卸载
func appstoreUninstall(c *gin.Context, st *state.AppState) error {
	if err := service.AppStoreUninstall(st, c.Param("key")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}

// appstoreUpgrade 升级
func appstoreUpgrade(c *gin.Context, st *state.AppState) error {
	if err := service.AppStoreUpgrade(st, c.Param("key")); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true})
	return nil
}
