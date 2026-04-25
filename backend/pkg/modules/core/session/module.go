package session

import (
	"context"
	authcontroller "homelab/pkg/controllers/core/auth"
	"homelab/pkg/controllers/middlewares"
	runtimepkg "homelab/pkg/runtime"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.session" }

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AuditMiddleware("rbac"))
			r.With(middlewares.RequirePermission("list", "rbac")).Get("/sessions", authcontroller.ScanSessionsHandler)
			r.With(middlewares.RequirePermission("admin", "rbac")).Delete("/sessions/{id}", authcontroller.RevokeSessionHandler)
		})
	})
}

func (m *Module) Start(context.Context) error { return nil }

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
