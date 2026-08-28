package api

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// Router 构建 gin 路由:public 组(登录)+ protected 组(全部业务)+ SPA 回退
// basePath:面板 URI 前缀(settings.webBasePath,默认 "/")
func Router(st *state.AppState, basePath string, static gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	deps := NewDeps(st)
	// 面板监听域名(webDomain):设置后仅该域名可访问,IP/其他域名访问返回 404;
	// 本机回环(localhost/127.0.0.1/::1)放行,便于本地运维与安装脚本调用 API
	r.Use(func(c *gin.Context) {
		domain := strings.TrimSpace(st.Settings.Get().WebDomain)
		if domain != "" {
			host := c.Request.Host
			if i := strings.Index(host, ":"); i >= 0 {
				host = host[:i]
			}
			lh := strings.ToLower(host)
			if lh != "localhost" && lh != "127.0.0.1" && lh != "::1" && !strings.EqualFold(lh, strings.ToLower(domain)) {
				c.AbortWithStatusJSON(404, gin.H{"error": "not found"})
				return
			}
		}
		c.Next()
	})

	if basePath == "" || basePath == "/" {
		basePath = ""
	} else {
		basePath = "/" + strings.Trim(basePath, "/")
	}

	api := r.Group(basePath + "/api")

	// ---------- public ----------
	api.GET("/health", H(systemHealth).Handler(deps))
	api.GET("/system/default-account", H(systemDefaultAccount).Handler(deps))
	api.GET("/system/public-config", H(systemPublicConfig).Handler(deps))
	api.GET("/system/wallpaper", H(wallpaperGet).Handler(deps))
	api.POST("/login", H(systemLogin).Handler(deps))
	api.POST("/login/totp", H(systemLoginTotp).Handler(deps))
	// 应用图标:公开(1Panel 同款,<img> 无 header 可带 token)
	api.GET("/apps/icon/:key", H(appstoreIcon).Handler(deps))

	// ---------- protected ----------
	p := api.Group("")
	p.Use(AuthMiddleware(st))

	p.GET("/me", H(systemMe).Handler(deps))
	p.POST("/profile", H(systemUpdateProfile).Handler(deps))
	p.POST("/avatar", H(systemUploadAvatar).Handler(deps))
	p.GET("/avatar/:file", H(systemServeAvatar).Handler(deps))
	p.POST("/system/wallpaper", H(wallpaperSave).Handler(deps))
	p.POST("/password", H(systemChangePassword).Handler(deps))
	p.POST("/totp/setup", H(systemTotpSetup).Handler(deps))
	p.POST("/totp/enable", H(systemTotpEnable).Handler(deps))
	p.POST("/totp/disable", H(systemTotpDisable).Handler(deps))

	p.GET("/system/info", H(systemInfo).Handler(deps))
	p.GET("/system/host", H(monitorHost).Handler(deps))
	p.GET("/system/monitor", H(monitorMonitor).Handler(deps))
	p.GET("/system/public-ip", H(systemPublicIP).Handler(deps))
	p.GET("/system/registry-mirrors", H(monitorRegistryMirrors).Handler(deps))
	p.PUT("/system/registry-mirrors", H(monitorSaveRegistryMirrors).Handler(deps))
	p.POST("/system/restart-docker", H(monitorRestartDocker).Handler(deps))

	// ---------- 面板设置 / 日志 / 配置 / 备份恢复 / 重启(仿 3x-ui) ----------
	p.GET("/system/settings", H(panelSettings).Handler(deps))
	p.PUT("/system/settings", H(panelSettingsSave).Handler(deps))
	p.GET("/system/logs", H(panelLogs).Handler(deps))
	p.GET("/system/events", H(panelEvents).Handler(deps))
	p.POST("/system/test-email", H(panelTestEmail).Handler(deps))
	p.GET("/system/config", H(panelConfig).Handler(deps))
	p.GET("/system/backup", H(panelBackup).Handler(deps))
	p.POST("/system/restore", H(panelRestore).Handler(deps))
	p.POST("/system/restart", H(panelRestart).Handler(deps))

	p.GET("/license", H(licenseGet).Handler(deps))
	p.POST("/license/activate", H(licenseActivate).Handler(deps))
	p.POST("/license/activate-file", H(licenseActivateFile).Handler(deps))
	p.POST("/license/deactivate", H(licenseDeactivate).Handler(deps))
	p.GET("/license/demo", H(licenseDemoKey).Handler(deps))

	// ---------- 在线更新 ----------
	p.GET("/update/check", H(updateCheck).Handler(deps))
	p.POST("/update/apply", H(updateApply).Handler(deps))

	p.GET("/ws/events", H(systemEventsWS).Handler(deps))

	p.GET("/apps", H(appstoreList).Handler(deps))
	p.GET("/apps/:key", H(appstoreDetail).Handler(deps))
	p.POST("/apps/sync", H(appstoreSync).Handler(deps))
	p.POST("/apps/:key/preview", H(appstorePreview).Handler(deps))
	p.POST("/apps/:key/install", H(appstoreInstall).Handler(deps))
	p.POST("/apps/:key/uninstall", H(appstoreUninstall).Handler(deps))
	p.POST("/apps/:key/upgrade", H(appstoreUpgrade).Handler(deps))

	p.GET("/containers", H(containersList).Handler(deps))
	p.POST("/containers", H(containersCreate).Handler(deps))
	p.POST("/containers/prune", H(containersPrune).Handler(deps))
	p.GET("/containers/:id", H(containersInspect).Handler(deps))
	p.DELETE("/containers/:id", H(containersRemove).Handler(deps))
	p.POST("/containers/:id/start", H(containersStart).Handler(deps))
	p.POST("/containers/:id/stop", H(containersStop).Handler(deps))
	p.POST("/containers/:id/restart", H(containersRestart).Handler(deps))
	p.POST("/containers/:id/recreate", H(containersRecreate).Handler(deps))
	p.POST("/containers/:id/kill", H(containersKill).Handler(deps))
	p.POST("/containers/:id/pause", H(containersPause).Handler(deps))
	p.POST("/containers/:id/unpause", H(containersUnpause).Handler(deps))
	p.POST("/containers/:id/rename", H(containersRename).Handler(deps))
	p.GET("/containers/:id/logs", H(containersLogsWS).Handler(deps))
	p.GET("/containers/:id/stats", H(containersStatsWS).Handler(deps))
	p.GET("/containers/:id/terminal", H(containersTerminalWS).Handler(deps))

	p.GET("/images", H(imagesList).Handler(deps))
	p.POST("/images/pull", H(imagesPull).Handler(deps))
	p.POST("/images/prune", H(imagesPrune).Handler(deps))
	p.GET("/images/:id", H(imagesInspect).Handler(deps))
	p.DELETE("/images/:id", H(imagesRemove).Handler(deps))
	p.POST("/images/:id/tag", H(imagesTag).Handler(deps))

	p.GET("/networks", H(networksList).Handler(deps))
	p.POST("/networks", H(networksCreate).Handler(deps))
	p.POST("/networks/prune", H(networksPrune).Handler(deps))
	p.GET("/networks/:id", H(networksInspect).Handler(deps))
	p.DELETE("/networks/:id", H(networksRemove).Handler(deps))

	p.GET("/volumes", H(volumesList).Handler(deps))
	p.POST("/volumes", H(volumesCreate).Handler(deps))
	p.POST("/volumes/prune", H(volumesPrune).Handler(deps))
	p.GET("/volumes/:name", H(volumesInspect).Handler(deps))
	p.DELETE("/volumes/:name", H(volumesRemove).Handler(deps))

	p.GET("/compose", H(composeList).Handler(deps))
	p.POST("/compose", H(composeUp).Handler(deps))
	p.POST("/compose/:project/adopt", H(composeAdopt).Handler(deps))
	p.GET("/compose/:project", H(composeInspect).Handler(deps))
	p.PUT("/compose/:project", H(composeUpdate).Handler(deps))
	p.DELETE("/compose/:project", H(composeRemove).Handler(deps))
	p.POST("/compose/:project/start", H(composeStart).Handler(deps))
	p.POST("/compose/:project/stop", H(composeStop).Handler(deps))
	p.POST("/compose/:project/restart", H(composeRestart).Handler(deps))
	p.POST("/compose/:project/down", H(composeDown).Handler(deps))
	p.GET("/compose/:project/logs", H(composeLogsWS).Handler(deps))

	r.NoRoute(static)
	return r
}
