package auth

import (
	"context"
	authcontroller "homelab/pkg/controllers/core/auth"
	"homelab/pkg/controllers/routerx"
	authrepo "homelab/pkg/repositories/core/auth"
	runtimepkg "homelab/pkg/runtime"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.auth" }

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	authrepo.Configure(deps.DB)
	return nil
}

func (m *Module) Routes() runtimepkg.RouteHandler {
	return routerx.New("/auth",
		routerx.Group("",
			routerx.Routes(
				routerx.Get("/ping", authcontroller.PingHandler),
				routerx.Post("/login", authcontroller.LoginHandler),
			),
		),
		routerx.Group("",
			routerx.WithScope(routerx.Scope{
				Resource: "auth",
				Audit:    "auth",
				UsesAuth: true,
			}),
			routerx.Routes(
				routerx.Get("/info", authcontroller.InfoHandler, "get"),
				routerx.Post("/logout", authcontroller.LogoutHandler, "update"),
				routerx.Get("/sessions", authcontroller.ScanSessionsHandler, "list"),
				routerx.Delete("/sessions/{id}", authcontroller.RevokeSessionHandler, "admin"),
			),
		),
	)
}

func (m *Module) Start(context.Context) error { return nil }

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
