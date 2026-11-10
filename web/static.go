package web

import (
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gin-gonic/gin"

	"sslplatform/internal/platform/response"
)

// RegisterStatic 注册前端资源。缺失的脚本和样式返回 404，页面路由回退到 index.html。
func RegisterStatic(engine *gin.Engine) error {
	sub, err := fs.Sub(Dist, "dist")
	if err != nil {
		return err
	}
	fileServer := http.FileServer(http.FS(sub))
	engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			response.Fail(c, http.StatusNotFound, "接口不存在")
			return
		}
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			response.Fail(c, http.StatusMethodNotAllowed, "页面只接受 GET")
			return
		}
		name := strings.TrimPrefix(c.Request.URL.Path, "/")
		if name == "" {
			name = "index.html"
		}
		file, err := sub.Open(name)
		if err != nil {
			if isAsset(name) {
				response.Fail(c, http.StatusNotFound, "资源不存在")
				return
			}
			data, readErr := fs.ReadFile(sub, "index.html")
			if readErr != nil {
				response.Fail(c, http.StatusNotFound, "页面不存在")
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
			return
		}
		_ = file.Close()
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
	return nil
}

func isAsset(name string) bool {
	switch strings.ToLower(path.Ext(name)) {
	case ".js", ".css", ".map", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".webp", ".woff", ".woff2", ".ttf":
		return true
	default:
		return false
	}
}
