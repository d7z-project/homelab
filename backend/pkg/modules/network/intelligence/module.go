package intelligence

import (
	"context"
	controllerdeps "homelab/pkg/controllers"
	"homelab/pkg/controllers/middlewares"
	intcontroller "homelab/pkg/controllers/network/intelligence"
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
	r.Route("/network/intelligence", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AuditMiddleware("network/intelligence"))
			r.Use(controllerdeps.WithIntelligenceControllerDeps(m.service))

			r.With(middlewares.RequirePermission("list", "network/intelligence")).Get("/sources", intcontroller.ScanIntelligenceSourcesHandler)
			r.With(middlewares.RequirePermission("create", "network/intelligence")).Post("/sources", intcontroller.CreateIntelligenceSourceHandler)
			r.With(middlewares.RequirePermission("update", "network/intelligence")).Put("/sources/{id}", intcontroller.UpdateIntelligenceSourceHandler)
			r.With(middlewares.RequirePermission("delete", "network/intelligence")).Delete("/sources/{id}", intcontroller.DeleteIntelligenceSourceHandler)
			r.With(middlewares.RequirePermission("execute", "network/intelligence")).Post("/sources/{id}/sync", intcontroller.SyncIntelligenceSourceHandler)
			r.With(middlewares.RequirePermission("execute", "network/intelligence")).Post("/sync/{id}/cancel", intcontroller.CancelIntelligenceSyncHandler)
		})
	})
}

func (m *Module) Start(ctx context.Context) error {
	intservice.RegisterDiscovery()
	return m.service.Init(ctx)
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
