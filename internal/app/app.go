package app

import (
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"sslplatform/internal/appinfo"
	"sslplatform/internal/platform/config"
	"sslplatform/internal/platform/database"
	"sslplatform/internal/platform/logging"
)

// Run 启动依赖注入容器和 HTTP 服务。
func Run() {
	if wantsVersion(os.Args[1:]) {
		fmt.Println(appinfo.Version)
		return
	}
	logging.Setup(nil)
	fx.New(
		Module,
		fx.WithLogger(func() fxevent.Logger { return logging.FxLogger{} }),
	).Run()
}

func announce(cfg *config.Config, _ database.Ready) {
	slog.Info("证枢启动",
		"version", appinfo.Version,
		"addr", logging.FormatAddr(cfg.Server.Addr),
		"database", cfg.Database.Path,
	)
	if cfg.UsingDefaultPassword() {
		slog.Warn("正在使用默认管理员密码 " + cfg.Seed.AdminPassword + "，请尽快在系统配置中修改，或通过环境变量 SSL_ADMIN_PASSWORD 设置")
	}
	if cfg.UsingDefaultSecret() {
		slog.Warn("正在使用默认 JWT 密钥，请尽快修改，或通过环境变量 SSL_JWT_SECRET 设置")
	}
}

func wantsVersion(args []string) bool {
	for _, arg := range args {
		if arg == "--version" || arg == "-version" {
			return true
		}
	}
	return false
}
