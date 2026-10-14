package common

import "go.uber.org/fx"

var Module = fx.Module(
	"common",
	fx.Provide(
		NewRepository,
		NewService,
		NewController,
	),
	fx.Invoke(RegisterRouter),
)
