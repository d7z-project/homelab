package audit

import (
	"context"
	auditcontroller "homelab/pkg/controllers/core/audit"
	"homelab/pkg/controllers/routerx"
	auditrepo "homelab/pkg/repositories/core/audit"
	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"
	auditservice "homelab/pkg/services/core/audit"
)

type Module struct{ registry *registryruntime.Registry }

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.audit" }

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	auditrepo.Configure(deps.DB)
	m.registry = deps.Registry
	return nil
}

func (m *Module) Routes() runtimepkg.RouteHandler {
	return routerx.New("/audit",
		routerx.WithScope(routerx.Scope{
			Resource: "audit",
			Audit:    "audit",
			UsesAuth: true,
		}),
		routerx.Routes(
			routerx.Get("/logs", auditcontroller.ScanAuditLogsHandler, "list"),
			routerx.Post("/logs/cleanup", auditcontroller.CleanupAuditLogsHandler, "delete"),
		),
	)
}

func (m *Module) Start(ctx context.Context) error {
	_ = ctx
	auditservice.RegisterDiscovery(m.registry)
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
