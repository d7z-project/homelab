package discovery

import (
	"context"
	discoverycontroller "homelab/pkg/controllers/core/discovery"
	"homelab/pkg/controllers/middlewares"
	runtimepkg "homelab/pkg/runtime"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.discovery" }

func (m *Module) Init(runtimepkg.ModuleDeps) error { return nil }

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/discovery", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			discoverycontroller.DiscoveryController(r)
		})
	})
}

func (m *Module) Start(context.Context) error { return nil }

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
