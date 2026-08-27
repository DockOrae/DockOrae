package api

import (
	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// Router 构建 gin 路由:public 组(登录)+ protected 组(全部业务)+ SPA 回退
func Router(st *state.AppState, static gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group("/api")

	// ---------- public ----------
	api.GET("/health", H(systemHealth).Handler(st))
	api.POST("/login", H(systemLogin).Handler(st))
	api.POST("/login/totp", H(systemLoginTotp).Handler(st))

	// ---------- protected ----------
	p := api.Group("")
	p.Use(AuthMiddleware(st))

	p.GET("/me", H(systemMe).Handler(st))
	p.POST("/profile", H(systemUpdateProfile).Handler(st))
	p.POST("/avatar", H(systemUploadAvatar).Handler(st))
	p.GET("/avatar/:file", H(systemServeAvatar).Handler(st))
	p.POST("/password", H(systemChangePassword).Handler(st))
	p.POST("/totp/setup", H(systemTotpSetup).Handler(st))
	p.POST("/totp/enable", H(systemTotpEnable).Handler(st))
	p.POST("/totp/disable", H(systemTotpDisable).Handler(st))

	p.GET("/system/info", H(systemInfo).Handler(st))
	p.GET("/system/host", H(monitorHost).Handler(st))
	p.GET("/system/monitor", H(monitorMonitor).Handler(st))
	p.GET("/system/registry-mirrors", H(monitorRegistryMirrors).Handler(st))
	p.PUT("/system/registry-mirrors", H(monitorSaveRegistryMirrors).Handler(st))
	p.POST("/system/restart-docker", H(monitorRestartDocker).Handler(st))

	p.GET("/terminal/quick-commands", H(terminalQuickCommands).Handler(st))
	p.POST("/terminal/quick-commands", H(terminalAddQuickCommand).Handler(st))
	p.DELETE("/terminal/quick-commands/:id", H(terminalDeleteQuickCommand).Handler(st))
	p.GET("/terminal/settings", H(terminalGetSettings).Handler(st))
	p.PUT("/terminal/settings", H(terminalSaveSettings).Handler(st))
	p.GET("/terminal/self", H(terminalSelfContainer).Handler(st))
	p.GET("/terminal/self/ws", H(terminalSelfWS).Handler(st))

	p.GET("/hosts", H(hostsList).Handler(st))
	p.POST("/hosts", H(hostsAdd).Handler(st))
	p.PUT("/hosts/:id", H(hostsUpdate).Handler(st))
	p.DELETE("/hosts/:id", H(hostsDelete).Handler(st))
	p.POST("/hosts/:id/test", H(hostsTest).Handler(st))
	p.GET("/hosts/:id/terminal", H(hostsTerminalWS).Handler(st))

	p.GET("/license", H(licenseGet).Handler(st))
	p.POST("/license/activate", H(licenseActivate).Handler(st))
	p.POST("/license/activate-file", H(licenseActivateFile).Handler(st))
	p.POST("/license/deactivate", H(licenseDeactivate).Handler(st))
	p.GET("/license/demo", H(licenseDemoKey).Handler(st))

	p.GET("/ws/events", H(systemEventsWS).Handler(st))

	p.GET("/containers", H(containersList).Handler(st))
	p.POST("/containers", H(containersCreate).Handler(st))
	p.POST("/containers/prune", H(containersPrune).Handler(st))
	p.GET("/containers/:id", H(containersInspect).Handler(st))
	p.DELETE("/containers/:id", H(containersRemove).Handler(st))
	p.POST("/containers/:id/start", H(containersStart).Handler(st))
	p.POST("/containers/:id/stop", H(containersStop).Handler(st))
	p.POST("/containers/:id/restart", H(containersRestart).Handler(st))
	p.POST("/containers/:id/kill", H(containersKill).Handler(st))
	p.POST("/containers/:id/pause", H(containersPause).Handler(st))
	p.POST("/containers/:id/unpause", H(containersUnpause).Handler(st))
	p.POST("/containers/:id/rename", H(containersRename).Handler(st))
	p.GET("/containers/:id/logs", H(containersLogsWS).Handler(st))
	p.GET("/containers/:id/stats", H(containersStatsWS).Handler(st))
	p.GET("/containers/:id/terminal", H(containersTerminalWS).Handler(st))

	p.GET("/images", H(imagesList).Handler(st))
	p.POST("/images/pull", H(imagesPull).Handler(st))
	p.POST("/images/prune", H(imagesPrune).Handler(st))
	p.GET("/images/:id", H(imagesInspect).Handler(st))
	p.DELETE("/images/:id", H(imagesRemove).Handler(st))
	p.POST("/images/:id/tag", H(imagesTag).Handler(st))

	p.GET("/networks", H(networksList).Handler(st))
	p.POST("/networks", H(networksCreate).Handler(st))
	p.POST("/networks/prune", H(networksPrune).Handler(st))
	p.GET("/networks/:id", H(networksInspect).Handler(st))
	p.DELETE("/networks/:id", H(networksRemove).Handler(st))

	p.GET("/volumes", H(volumesList).Handler(st))
	p.POST("/volumes", H(volumesCreate).Handler(st))
	p.POST("/volumes/prune", H(volumesPrune).Handler(st))
	p.GET("/volumes/:name", H(volumesInspect).Handler(st))
	p.DELETE("/volumes/:name", H(volumesRemove).Handler(st))

	p.GET("/compose", H(composeList).Handler(st))
	p.POST("/compose", H(composeUp).Handler(st))
	p.GET("/compose/:project", H(composeInspect).Handler(st))
	p.PUT("/compose/:project", H(composeUpdate).Handler(st))
	p.DELETE("/compose/:project", H(composeRemove).Handler(st))
	p.POST("/compose/:project/start", H(composeStart).Handler(st))
	p.POST("/compose/:project/stop", H(composeStop).Handler(st))
	p.POST("/compose/:project/restart", H(composeRestart).Handler(st))
	p.POST("/compose/:project/down", H(composeDown).Handler(st))
	p.GET("/compose/:project/logs", H(composeLogsWS).Handler(st))

	r.NoRoute(static)
	return r
}
