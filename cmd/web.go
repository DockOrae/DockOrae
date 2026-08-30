package cmd

import (
	"mime"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/DockOrae/DockOrae/public"
)

// serveStatic 静态资源服务:SPA 路由回退到 index.html(basePath 为 URI 前缀)
func serveStatic(basePath string) gin.HandlerFunc {
	if basePath == "" || basePath == "/" {
		basePath = ""
	}
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		// 安全入口(webBasePath 非 /):非入口前缀的请求 302 到入口(保留路径),
		// 使页面与 API 都能带前缀到达;静态资源(/assets、logo)直接服务不要求前缀
		if basePath != "" {
			if strings.HasPrefix(p, "/assets/") || p == "/logo.svg" || p == "/favicon.ico" {
				// 直接服务(不剥离前缀)
			} else if !strings.HasPrefix(p, basePath) {
				c.Redirect(302, basePath+p)
				return
			} else {
				p = strings.TrimPrefix(p, basePath)
			}
		}
		// 未知 /api 路径返回 404,其余交给 SPA
		if strings.HasPrefix(p, "/api/") || p == "/api" {
			c.JSON(404, gin.H{"error": "not found"})
			return
		}

		path := strings.TrimPrefix(p, "/")
		rel := "dist/" + path

		var (
			data []byte
			err  error
			name string
		)
		if strings.HasPrefix(path, "assets/") {
			// 带 hash 的资源:直接命中,不做 SPA 回退
			name = rel
			data, err = public.Dist.ReadFile(rel)
			if err != nil {
				c.Status(404)
				return
			}
		} else {
			name = rel
			data, err = public.Dist.ReadFile(rel)
			if err != nil {
				// SPA 回退
				name = "dist/index.html"
				data, err = public.Dist.ReadFile(name)
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
