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

// serveStatic 静态资源服务:SPA 路由回退到 index.html(basePath 为 URI 前缀)
func serveStatic(basePath string) gin.HandlerFunc {
	if basePath == "" || basePath == "/" {
		basePath = ""
	}
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		// 带前缀的部署:非 basePath 请求跳转到 basePath
		if basePath != "" {
			if !strings.HasPrefix(p, basePath) {
				c.Redirect(302, basePath)
				return
			}
			p = strings.TrimPrefix(p, basePath)
		}
		// 未知 /api 路径返回 404,其余交给 SPA
		if strings.HasPrefix(p, "/api/") || p == "/api" {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}

		path := strings.TrimPrefix(p, "/")
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
}
