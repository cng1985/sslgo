package datasource

import "go.uber.org/fx"

var Module = fx.Module(
	"datasource",
	fx.Provide(
		NewRepository,
		NewService,
		NewController,
	),
	fx.Invoke(RegisterRouter),
)
