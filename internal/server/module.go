package server

import (
	"net/http"

	"go.uber.org/fx"
)

var Module = fx.Module(
	"server",
	fx.Provide(
		NewEndpoint,
		NewEngine,
		NewRegistrar,
		ProvideRouter,
		NewHTTPServer,
	),
	fx.Invoke(RegisterSystem, func(*http.Server) {}),
)
