package etl

import "go.uber.org/fx"

var Module = fx.Module(
	"etl",
	fx.Provide(
		NewRepository,
		NewIssuer,
		NewDeployer,
		NewETLService,
		NewController,
	),
	fx.Invoke(
		RegisterRouter,
	),
)
