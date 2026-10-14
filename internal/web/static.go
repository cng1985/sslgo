package web

import (
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"sslplatform/internal/common/response"
)

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
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}
		if _, err := sub.Open(path); err != nil {
			data, readErr := fs.ReadFile(sub, "index.html")
			if readErr != nil {
				response.Fail(c, http.StatusNotFound, "页面不存在")
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
			return
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
	return nil
}
