package auth

import (
	"context"
	"homelab/pkg/routes/core"
	runtimepkg "homelab/pkg/runtime"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.auth" }

func (m *Module) RegisterRoutes(r chi.Router) { core.RegisterAuth(r) }

func (m *Module) Start(context.Context) error { return nil }

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
