package web

// 前端静态资源（Next.js output: export），构建时由 scripts/embed-frontend.sh 填充。
// Next 静态导出布局：/login → login.html；/ → index.html；资源在 _next/、assets/ 等。
import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed all:frontend
var frontendFS embed.FS

// Register 静态前端路由（Next export：path.html 映射 + SPA fallback）
func Register(r *gin.Engine, apiPrefix string) {
	sub, err := fs.Sub(frontendFS, "frontend")
	if err != nil {
		return
	}

	r.NoRoute(func(c *gin.Context) {
		// API 前缀 404 保持 JSON
		if strings.HasPrefix(c.Request.URL.Path, apiPrefix) {
			c.JSON(http.StatusNotFound, gin.H{"statusCode": 404, "message": "接口不存在"})
			return
		}

		p := strings.TrimPrefix(c.Request.URL.Path, "/")
		p = strings.TrimSuffix(p, "/")

		// 依次尝试：精确路径 → path.html（Next 导出布局）；SPA fallback → index.html
		candidates := []string{p, p + ".html"}
		if p == "" {
			candidates = []string{"index.html"}
		}
		for _, cand := range candidates {
			if cand == "" {
				continue
			}
			if f, err := sub.Open(cand); err == nil {
				f.Close()
				serveFile(c, sub, cand)
				return
			}
		}
		// 带扩展名的资源（js/css/png）没找到就真 404
		if strings.Contains(p, ".") {
			c.Status(http.StatusNotFound)
			return
		}
		// SPA 路由 fallback 到首页
		serveFile(c, sub, "index.html")
	})
}

func serveFile(c *gin.Context, sub fs.FS, name string) {
	data, err := fs.ReadFile(sub, name)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	ctype := mimeByExt(name)
	c.Header("Content-Type", ctype)
	// 带内容 hash 的 _next 静态资源长缓存；HTML 不缓存
	if strings.HasPrefix(name, "_next/static/") {
		c.Header("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		c.Header("Cache-Control", "no-cache")
	}
	c.Data(http.StatusOK, ctype, data)
}

func mimeByExt(name string) string {
	switch {
	case strings.HasSuffix(name, ".html"):
		return "text/html; charset=utf-8"
	case strings.HasSuffix(name, ".js"):
		return "text/javascript; charset=utf-8"
	case strings.HasSuffix(name, ".css"):
		return "text/css; charset=utf-8"
	case strings.HasSuffix(name, ".json"):
		return "application/json"
	case strings.HasSuffix(name, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	case strings.HasSuffix(name, ".jpg"), strings.HasSuffix(name, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(name, ".woff2"):
		return "font/woff2"
	case strings.HasSuffix(name, ".woff"):
		return "font/woff"
	case strings.HasSuffix(name, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(name, ".txt"):
		return "text/plain; charset=utf-8"
	default:
		return "application/octet-stream"
	}
}
