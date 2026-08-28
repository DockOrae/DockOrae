package api

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/MinimaxFlora/Docker_Manager_Go/internal/auth"
	"github.com/MinimaxFlora/Docker_Manager_Go/internal/state"
)

// AuthMiddleware token 从 Authorization 头或 ?token= query 取(WS 用);
// 未认证响应按 noAuthSetting(仿 1Panel:200 帮助页 / 400 / 403 / 404 / 408 / 416 / 444 / 500)
func AuthMiddleware(st *state.AppState) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := auth.BearerToken(c.GetHeader("Authorization"))
		if token == "" {
			token = auth.QueryToken(c.Request.URL.Query())
		}
		username, ok := auth.VerifyToken(st.Cfg.JWTSecret, token)
		if !ok {
			noAuthRespond(c, st)
			return
		}
		c.Set("username", username)
		c.Next()
	}
}

// noAuthCode 未认证时的响应状态码(仿 1Panel noAuthSetting)
func noAuthCode(st *state.AppState) int {
	s := st.Settings.Get()
	switch s.NoAuthSetting {
	case "200", "400", "401", "403", "404", "408", "416", "444", "500":
		n, _ := strconv.Atoi(s.NoAuthSetting)
		return n
	default:
		return 401
	}
}

// noAuthRespond 按未认证设置返回响应
func noAuthRespond(c *gin.Context, st *state.AppState) {
	code := noAuthCode(st)
	if code == 200 {
		// 200:返回帮助页(1Panel 语义:让扫描器以为服务正常)
		c.Header("Content-Type", "text/html; charset=utf-8")
		c.String(200, "<!DOCTYPE html><html><head><title>Docker Manager</title></head>"+
			"<body><h1>Docker Manager</h1><p>Go 编写的 Docker 管理面板</p></body></html>")
		return
	}
	if code == 444 {
		// 444:nginx 风格"直接关闭连接"
		c.Header("Connection", "close")
		c.AbortWithStatusJSON(444, gin.H{"error": "unauthorized"})
		return
	}
	c.AbortWithStatusJSON(code, gin.H{"error": "未登录或登录已过期"})
}
