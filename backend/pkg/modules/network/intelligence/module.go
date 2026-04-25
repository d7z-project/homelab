package intelligence

import (
	"context"
	"homelab/pkg/routes"
	runtimepkg "homelab/pkg/runtime"
	intservice "homelab/pkg/services/network/intelligence"
	ipservice "homelab/pkg/services/network/ip"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	service *intservice.IntelligenceService
}

func New(enricher *ipservice.MMDBManager) *Module {
	return &Module{service: intservice.NewIntelligenceService(enricher)}
}

func (m *Module) Name() string { return "network.intelligence" }

func (m *Module) RegisterRoutes(r chi.Router) {
	routes.RegisterNetworkIntelligence(r, m.service)
}

func (m *Module) Start(ctx context.Context) error {
	intservice.RegisterDiscovery()
	return m.service.Init(ctx)
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
