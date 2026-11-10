package app

import (
	"go.uber.org/fx"

	"sslplatform/internal/modules/certificate"
	"sslplatform/internal/modules/datasource"
	"sslplatform/internal/modules/query"
	"sslplatform/internal/modules/schedule"
	"sslplatform/internal/modules/task"
	"sslplatform/internal/platform/auth"
	"sslplatform/internal/platform/config"
	"sslplatform/internal/platform/database"
	"sslplatform/internal/platform/logging"
	"sslplatform/internal/platform/middleware"
	"sslplatform/internal/platform/server"
	"sslplatform/web"
)

// Module 组合平台、业务模块和 Web。
var Module = fx.Module(
	"app",
	config.Module,
	database.Module,
	fx.Provide(Bootstrap),
	loggingInvoke(),
	auth.Module,
	middleware.Module,
	datasource.Module,
	task.Module,
	certificate.Module,
	schedule.Module,
	query.Module,
	web.Module,
	server.Module,
	fx.Invoke(announce),
)

func loggingInvoke() fx.Option {
	return fx.Invoke(func(cfg *config.Config) { logging.Apply(cfg) })
}
