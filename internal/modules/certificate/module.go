package certificate

import "go.uber.org/fx"

var Module = fx.Module(
	"etl",
	fx.Provide(
		NewRepository,
		NewIssuer,
		NewDeployer,
		NewResource,
		NewController,
	),
	fx.Invoke(
		RegisterRouter,
	),
)
