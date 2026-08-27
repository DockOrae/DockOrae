package api

import (
	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/auth"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// AuthMiddleware token 从 Authorization 头或 ?token= query 取(WS 用)
func AuthMiddleware(st *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := auth.BearerToken(c.GetHeader("Authorization"))
		if token == "" {
			token = auth.QueryToken(c.Request.URL.Query())
		}
		username, ok := auth.VerifyToken(st.Cfg.JWTSecret, token)
		if !ok {
			c.AbortWithStatusJSON(401, gin.H{"error": "未登录或登录已过期"})
			return
		}
		c.Set("username", username)
		c.Next()
	}
}
