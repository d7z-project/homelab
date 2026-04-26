package secret

import (
	"context"
	runtimepkg "homelab/pkg/runtime"
	secretservice "homelab/pkg/services/core/secret"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.secret" }

func (m *Module) Init(runtimepkg.ModuleDeps) error { return nil }

func (m *Module) RegisterRoutes(chi.Router) {}

func (m *Module) Start(context.Context) error { return secretservice.ValidateConfig() }

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
