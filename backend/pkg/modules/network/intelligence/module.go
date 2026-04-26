package intelligence

import (
	"context"
	"net/http"

	controllerdeps "homelab/pkg/controllers"
	intcontroller "homelab/pkg/controllers/network/intelligence"
	"homelab/pkg/controllers/routerx"
	runtimepkg "homelab/pkg/runtime"
	intservice "homelab/pkg/services/network/intelligence"
	ipservice "homelab/pkg/services/network/ip"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	enricher *ipservice.MMDBManager
	service  *intservice.IntelligenceService
}

func New(enricher *ipservice.MMDBManager) *Module {
	return &Module{enricher: enricher}
}

func (m *Module) Name() string { return "network.intelligence" }

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	m.service = intservice.NewIntelligenceService(deps, m.enricher)
	return nil
}

func (m *Module) RegisterRoutes(r chi.Router) {
	routerx.Mount(r, "/network/intelligence", routerx.Scope{
		Resource: "network/intelligence",
		Audit:    "network/intelligence",
		UsesAuth: true,
		Extra: []func(http.Handler) http.Handler{
			controllerdeps.WithIntelligenceControllerDeps(m.service),
		},
	},
		routerx.Get("/sources", intcontroller.ScanIntelligenceSourcesHandler, "list"),
		routerx.Post("/sources", intcontroller.CreateIntelligenceSourceHandler, "create"),
		routerx.Put("/sources/{id}", intcontroller.UpdateIntelligenceSourceHandler, "update"),
		routerx.Delete("/sources/{id}", intcontroller.DeleteIntelligenceSourceHandler, "delete"),
		routerx.Post("/sources/{id}/sync", intcontroller.SyncIntelligenceSourceHandler, "execute"),
		routerx.Post("/sync/{id}/cancel", intcontroller.CancelIntelligenceSyncHandler, "execute"),
	)
}

func (m *Module) Start(ctx context.Context) error {
	intservice.RegisterDiscovery(runtimepkg.RegistryFromContext(ctx))
	return m.service.Init(ctx)
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
