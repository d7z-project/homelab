package rbac

import (
	"context"
	"homelab/pkg/routes/core"
	runtimepkg "homelab/pkg/runtime"
	rbacservice "homelab/pkg/services/core/rbac"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.rbac" }

func (m *Module) RegisterRoutes(r chi.Router) { core.RegisterRBAC(r) }

func (m *Module) Start(context.Context) error {
	rbacservice.RegisterDiscovery()
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
