package intelligence

import (
	"context"
	"net/http"

	intcontroller "homelab/pkg/controllers/network/intelligence"
	"homelab/pkg/controllers/routerx"
	intrepo "homelab/pkg/repositories/network/intelligence"
	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"
	intservice "homelab/pkg/services/network/intelligence"
	ipservice "homelab/pkg/services/network/ip"
)

type Module struct {
	enricher *ipservice.MMDBManager
	service  *intservice.IntelligenceService
	registry *registryruntime.Registry
}

func New(enricher *ipservice.MMDBManager) *Module {
	return &Module{enricher: enricher}
}

func (m *Module) Name() string { return "network.intelligence" }

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	intrepo.Configure(deps.DB)
	m.service = intservice.NewIntelligenceService(deps, m.enricher)
	m.registry = deps.Registry
	return nil
}

func (m *Module) Routes() runtimepkg.RouteHandler {
	return routerx.New("/network/intelligence",
		routerx.WithScope(routerx.Scope{
			Resource: "network/intelligence",
			Audit:    "network/intelligence",
			UsesAuth: true,
			Extra: []func(http.Handler) http.Handler{
				intcontroller.WithControllerDeps(m.service),
			},
		}),
		routerx.Routes(
			routerx.Get("/sources", intcontroller.ScanIntelligenceSourcesHandler, "list"),
			routerx.Post("/sources", intcontroller.CreateIntelligenceSourceHandler, "create"),
			routerx.Put("/sources/{id}", intcontroller.UpdateIntelligenceSourceHandler, "update"),
			routerx.Delete("/sources/{id}", intcontroller.DeleteIntelligenceSourceHandler, "delete"),
			routerx.Post("/sources/{id}/sync", intcontroller.SyncIntelligenceSourceHandler, "execute"),
			routerx.Post("/sync/{id}/cancel", intcontroller.CancelIntelligenceSyncHandler, "execute"),
		),
	)
}

func (m *Module) Start(ctx context.Context) error {
	_ = ctx
	intservice.RegisterDiscovery(m.registry)
	return m.service.Init(ctx)
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
