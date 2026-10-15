package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"sslplatform/internal/appinfo"
	"sslplatform/internal/common/response"
	"sslplatform/internal/config"
	"sslplatform/internal/httpapi"
	"sslplatform/internal/logging"
)

func NewHTTPServer(lc fx.Lifecycle, cfg *config.Config, engine *gin.Engine, ep *Endpoint) *http.Server {
	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			ln, err := net.Listen("tcp", srv.Addr)
			if err != nil {
				return err
			}
			ep.Set(ln.Addr().String())
			slog.Info("HTTP 服务已启动", "addr", logging.FormatAddr(ln.Addr().String()))
			go func() {
				if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
					slog.Error("http server", "err", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			return srv.Shutdown(ctx)
		},
	})
	return srv
}

func RegisterSystem(r *Registrar, cfg *config.Config) {
	r.Public(httpapi.Meta{
		Resource: "system", Action: "info", Path: "/system/info",
		Summary: "服务健康信息，不包含密钥",
	}, func(c *gin.Context) {
		response.OK(c, map[string]any{
			"name":     "sslplatform",
			"version":  appinfo.Version,
			"api":      "/api/v1",
			"method":   "POST",
			"acme":     cfg.ACME.DirectoryURL,
			"response": map[string]string{"code": "业务码，成功为 200", "msg": "说明", "data": "结果"},
		})
	})
	r.Protected(httpapi.Meta{
		Resource: "system", Action: "catalog", Path: "/system/catalog",
		Summary: "返回全部资源、动作和参数说明，供 AI Agent 发现接口",
	}, func(c *gin.Context) {
		response.OK(c, map[string]any{
			"version": appinfo.Version,
			"rules": []string{
				"所有接口使用 POST 和 JSON",
				"成功响应 code 为 200",
				"分页响应包含 page、size、total、responseType=page",
				"租户来自登录态，请求体禁止出现 tenantId",
			},
			"routes": r.Routes(),
		})
	})
}
