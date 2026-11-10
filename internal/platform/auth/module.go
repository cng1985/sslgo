package auth

import (
	"go.uber.org/fx"

	"sslplatform/internal/platform/middleware"
)

var Module = fx.Module(
	"auth",
	fx.Provide(
		NewRepository,
		NewResource,
		NewController,
		func(resource *Resource) middleware.Identity { return resource },
	),
	fx.Invoke(RegisterRouter),
)
