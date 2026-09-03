package middleware

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// Gzip 压缩响应（>=1KB 且 Accept-Encoding 允许时）。
// 用自定义实现而非 gin-contrib/gzip：零依赖、只压缩 JSON/HTML/JS/CSS。
type gzipWriter struct {
	gin.ResponseWriter
	zw *gzip.Writer
}

func (w *gzipWriter) Write(data []byte) (int, error) {
	return w.zw.Write(data)
}

func (w *gzipWriter) WriteString(s string) (int, error) {
	return w.zw.Write([]byte(s))
}

func shouldCompress(c *gin.Context) bool {
	if !strings.Contains(c.GetHeader("Accept-Encoding"), "gzip") {
		return false
	}
	reqMethod := c.Request.Method
	if reqMethod == http.MethodHead {
		return false
	}
	// 只压可压缩类型（JSON API 响应、HTML、静态资源）
	ct := c.Writer.Header().Get("Content-Type")
	ct = strings.ToLower(strings.TrimSpace(strings.Split(ct, ";")[0]))
	switch {
	case ct == "",
		strings.HasPrefix(ct, "application/json"),
		strings.HasPrefix(ct, "text/"),
		strings.HasPrefix(ct, "application/javascript"),
		strings.HasPrefix(ct, "image/svg+xml"):
		return true
	}
	return false
}

const minCompressSize = 1024

// Gzip 中间件：小响应不值得压缩开销，用 buffer 阈值判断
func Gzip() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !shouldCompress(c) {
			c.Next()
			return
		}

		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		c.Writer = &gzipWriter{c.Writer, gz}
		c.Header("Content-Encoding", "gzip")
		// 代理层（NGINX 等）不要再动
		c.Header("Vary", "Accept-Encoding")
		// 已设 immutable 长缓存的资源体积大，交由代理/CDN 处理；这里去掉 Content-Length（会变）
		c.Header("Content-Length", "")

		c.Next()
	}
}

var _ io.Writer = (*gzipWriter)(nil)
