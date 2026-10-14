package task

import "go.uber.org/fx"

var Module = fx.Module(
	"task",
	fx.Provide(
		NewRepository,
		NewService,
		NewController,
	),
	fx.Invoke(RegisterRouter),
)
