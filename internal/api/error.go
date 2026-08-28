package api

import (
	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// Deps 请求处理依赖集合:AppState + 具体 Service 实例。
// handler 通过 d.Containers.Create(...) 等直接调用具体服务,不再经包级兼容函数。
type Deps struct {
	St         *state.AppState
	Containers *service.ContainerService
	Images     *service.ImageService
	Networks   *service.NetworkService
	Volumes    *service.VolumeService
	Compose    *service.ComposeService
	AppStore   *service.AppStoreService
}

// NewDeps 从 AppState 构建具体服务集合(一次构建,全局复用)
func NewDeps(st *state.AppState) *Deps {
	return &Deps{
		St:         st,
		Containers: service.NewContainerService(st),
		Images:     service.NewImageService(st),
		Networks:   service.NewNetworkService(st),
		Volumes:    service.NewVolumeService(st),
		Compose:    service.NewComposeService(st),
		AppStore:   service.NewAppStoreService(st),
	}
}

// H 适配器:handler 返回 error,统一转 {error: message} JSON(注入 Deps)
type H func(c *gin.Context, d *Deps) error

func (h H) Handler(d *Deps) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h(c, d); err != nil {
			ae := service.AsApiError(err)
			c.AbortWithStatusJSON(ae.Status, gin.H{"error": ae.Message})
		}
	}
}

// parseBool 查询参数布尔解析
func parseBool(s string, def bool) bool {
	if s == "true" || s == "1" {
		return true
	}
	if s == "false" || s == "0" {
		return false
	}
	return def
}
