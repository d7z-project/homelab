package audit

import (
	"context"
	auditcontroller "homelab/pkg/controllers/core/audit"
	"homelab/pkg/controllers/routerx"
	runtimepkg "homelab/pkg/runtime"
	auditservice "homelab/pkg/services/core/audit"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.audit" }

func (m *Module) Init(runtimepkg.ModuleDeps) error { return nil }

func (m *Module) RegisterRoutes(r chi.Router) {
	routerx.Mount(r, "/audit", routerx.Scope{
		Resource: "audit",
		Audit:    "audit",
		UsesAuth: true,
	},
		routerx.Get("/logs", auditcontroller.ScanAuditLogsHandler, "list"),
		routerx.Post("/logs/cleanup", auditcontroller.CleanupAuditLogsHandler, "delete"),
	)
}

func (m *Module) Start(ctx context.Context) error {
	auditservice.RegisterDiscovery(runtimepkg.RegistryFromContext(ctx))
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
