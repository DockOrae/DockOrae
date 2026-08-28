package api

import (
	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/service"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// H 适配器:handler 返回 error,统一转 {error: message} JSON(注入 AppState)
type H func(c *gin.Context, st *state.AppState) error

func (h H) Handler(st *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := h(c, st); err != nil {
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
