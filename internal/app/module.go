package app

import (
	"go.uber.org/fx"

	"sslplatform/internal/auth"
	"sslplatform/internal/common"
	"sslplatform/internal/config"
	"sslplatform/internal/database"
	"sslplatform/internal/datasource"
	"sslplatform/internal/etl"
	"sslplatform/internal/scheduler"
	"sslplatform/internal/server"
	"sslplatform/internal/task"
	"sslplatform/internal/web"
)

// Module 组装全部 Fx 模块。main 只负责启动。
var Module = fx.Module(
	"app",
	config.Module,
	database.Module,
	auth.Module,
	datasource.Module,
	task.Module,
	etl.Module,
	scheduler.Module,
	common.Module,
	web.Module,
	server.Module,
)
