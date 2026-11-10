package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"sslplatform/internal/platform/config"
	"sslplatform/internal/platform/database"
	"sslplatform/internal/platform/logging"
)

func NewHTTPServer(lc fx.Lifecycle, cfg *config.Config, engine *gin.Engine, ep *Endpoint, shutdown fx.Shutdowner, _ database.Ready, _ struct {
	fx.In
	Before []BeforeHTTP `group:"beforeHTTP"`
}) *http.Server {
	srv := &http.Server{
		Addr:              cfg.Server.Addr,
		Handler:           engine,
		ReadHeaderTimeout: cfg.Server.ReadHeaderTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
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
					slog.Error("HTTP 服务异常退出", "err", err)
					_ = shutdown.Shutdown()
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopCtx, cancel := context.WithTimeout(ctx, cfg.Server.ShutdownTimeout)
			defer cancel()
			return srv.Shutdown(stopCtx)
		},
	})
	return srv
}
