package api

import (
	"github.com/gin-gonic/gin"

	"github.com/DockerManger/Docker_Manager_Go/internal/service"
)

// updateCheck 检测 GitHub 最新 Release(结果缓存 10 分钟,防 GitHub API 限流;失败不缓存)
func updateCheck(c *gin.Context, d *Deps) error {
	info, err := service.UpdateCheck(d.St, c.Request.Context())
	if err != nil {
		return err
	}
	c.JSON(200, info)
	return nil
}

// updateStatus 更新进度(前端轮询:downloading → extracting → replacing → restarting / pulling → helper → done)
func updateStatus(c *gin.Context, d *Deps) error {
	c.JSON(200, service.GetUpdateStatus())
	return nil
}

// updateApply 一键更新(异步启动,立即返回;进度经 /update/status 轮询)
func updateApply(c *gin.Context, d *Deps) error {
	if err := service.UpdateApply(d.St, c.Request.Context()); err != nil {
		return err
	}
	c.JSON(200, gin.H{"ok": true, "message": "update.started"})
	return nil
}
