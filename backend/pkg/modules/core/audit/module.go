package audit

import (
	"context"
	auditcontroller "homelab/pkg/controllers/core/audit"
	"homelab/pkg/controllers/middlewares"
	runtimepkg "homelab/pkg/runtime"
	auditservice "homelab/pkg/services/core/audit"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.audit" }

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/audit", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AuditMiddleware("audit"))
			r.With(middlewares.RequirePermission("list", "audit")).Get("/logs", auditcontroller.ScanAuditLogsHandler)
			r.With(middlewares.RequirePermission("delete", "audit")).Post("/logs/cleanup", auditcontroller.CleanupAuditLogsHandler)
		})
	})
}

func (m *Module) Start(context.Context) error {
	auditservice.RegisterDiscovery()
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
