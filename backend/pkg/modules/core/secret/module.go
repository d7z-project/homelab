package secret

import (
	"context"
	secretrepo "homelab/pkg/repositories/core/secret"
	runtimepkg "homelab/pkg/runtime"
	secretservice "homelab/pkg/services/core/secret"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.secret" }

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	secretrepo.Configure(deps.DB)
	return nil
}

func (m *Module) Routes() runtimepkg.RouteHandler { return nil }

func (m *Module) Start(context.Context) error { return secretservice.ValidateConfig() }

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
