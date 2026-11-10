package query

import "go.uber.org/fx"

var Module = fx.Module(
	"common",
	fx.Provide(
		NewRepository,
		NewResource,
		NewController,
	),
	fx.Invoke(RegisterRouter),
)
