package main

import (
	"embed"
	"mime"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:web/dist
var dist embed.FS

// serveStatic 静态资源服务:SPA 路由回退到 index.html
func serveStatic(c *gin.Context) {
	// 未知 /api 路径返回 404(与旧版 axum nest 行为一致),其余交给 SPA
	if strings.HasPrefix(c.Request.URL.Path, "/api/") || c.Request.URL.Path == "/api" {
		c.JSON(404, gin.H{"error": "not found"})
		return
	}

	path := strings.TrimPrefix(c.Request.URL.Path, "/")
	rel := "web/dist/" + path

	var (
		data []byte
		err  error
		name string
	)
	if strings.HasPrefix(path, "assets/") {
		// 带 hash 的资源:直接命中,不做 SPA 回退
		name = rel
		data, err = dist.ReadFile(rel)
		if err != nil {
			c.Status(404)
			return
		}
	} else {
		name = rel
		data, err = dist.ReadFile(rel)
		if err != nil {
			// SPA 回退
			name = "web/dist/index.html"
			data, err = dist.ReadFile(name)
			if err != nil {
				c.Status(404)
				return
			}
		}
	}

	ctype := mime.TypeByExtension(filepath.Ext(name))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	if strings.HasPrefix(path, "assets/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "no-cache")
	}
	c.Data(200, ctype, data)
}
