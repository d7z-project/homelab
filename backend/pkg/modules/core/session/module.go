package session

import (
	"context"
	authcontroller "homelab/pkg/controllers/core/auth"
	"homelab/pkg/controllers/routerx"
	runtimepkg "homelab/pkg/runtime"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.session" }

func (m *Module) RegisterRoutes(r chi.Router) {
	routerx.WithScope(r, routerx.Scope{
		Resource: "rbac",
		Audit:    "rbac",
		UsesAuth: true,
	},
		routerx.Get("/auth/sessions", authcontroller.ScanSessionsHandler, "list"),
		routerx.Delete("/auth/sessions/{id}", authcontroller.RevokeSessionHandler, "admin"),
	)
}

func (m *Module) Start(context.Context) error { return nil }

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
