package schedule

import (
	"go.uber.org/fx"

	"sslplatform/internal/platform/server"
)

var Module = fx.Module(
	"scheduler",
	fx.Provide(
		NewRepository,
		NewResource,
		NewController,
		fx.Annotate(
			func(*Resource) server.BeforeHTTP { return server.StartupOrder{} },
			fx.ResultTags(`group:"beforeHTTP"`),
		),
	),
	fx.Invoke(RegisterRouter),
)
