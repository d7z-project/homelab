package ip

import (
	"context"
	"net/http"

	controllerdeps "homelab/pkg/controllers"
	ipcontroller "homelab/pkg/controllers/network/ip"
	"homelab/pkg/controllers/routerx"
	runtimepkg "homelab/pkg/runtime"
	ipservice "homelab/pkg/services/network/ip"
	ruleservice "homelab/pkg/services/rules"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	enricher *ipservice.MMDBManager
	service  *ipservice.IPPoolService
	analysis *ipservice.AnalysisEngine
	exports  *ipservice.ExportManager
}

func New(enricher *ipservice.MMDBManager) *Module {
	return &Module{enricher: enricher}
}

func (m *Module) Name() string { return "network.ip" }

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	m.analysis = ipservice.NewAnalysisEngine(m.enricher)
	m.exports = ipservice.NewExportManager(deps, m.analysis)
	m.service = ipservice.NewIPPoolService(deps, m.analysis, m.exports)
	return nil
}

func (m *Module) RegisterRoutes(r chi.Router) {
	routerx.Mount(r, "/network/ip", routerx.Scope{
		Resource: "network/ip",
		Audit:    "network/ip",
		UsesAuth: true,
		Extra: []func(http.Handler) http.Handler{
			controllerdeps.WithIPControllerDeps(m.service, m.analysis, m.exports),
		},
	},
		routerx.Get("/pools", ipcontroller.ScanGroupsHandler, "list"),
		routerx.Post("/pools", ipcontroller.CreateGroupHandler, "create"),
		routerx.Put("/pools/{id}", ipcontroller.UpdateGroupHandler, "update"),
		routerx.Delete("/pools/{id}", ipcontroller.DeleteGroupHandler, "delete"),
		routerx.Get("/pools/{id}/preview", ipcontroller.PreviewPoolHandler, "get"),
		routerx.Post("/pools/{id}/entries", ipcontroller.ManagePoolEntryHandler, "update"),
		routerx.Delete("/pools/{id}/entries", ipcontroller.DeletePoolEntryHandler, "update"),
		routerx.Post("/analysis/hit-test", ipcontroller.HitTestHandler, "execute"),
		routerx.Get("/analysis/info", ipcontroller.IPInfoHandler, "get"),
		routerx.Get("/exports", ipcontroller.ScanExportsHandler, "list"),
		routerx.Get("/exports/tasks", ipcontroller.ScanExportTasksHandler, "list"),
		routerx.Post("/exports", ipcontroller.CreateExportHandler, "create"),
		routerx.Put("/exports/{id}", ipcontroller.UpdateExportHandler, "update"),
		routerx.Delete("/exports/{id}", ipcontroller.DeleteExportHandler, "delete"),
		routerx.Post("/exports/{id}/trigger", ipcontroller.TriggerIPExportHandler, "execute"),
		routerx.Get("/exports/task/{taskId}", ipcontroller.ExportTaskStatusHandler, "get"),
		routerx.Post("/exports/task/{taskId}/cancel", ipcontroller.CancelExportTaskHandler, "execute"),
		routerx.Get("/exports/download/{taskId}", ipcontroller.DownloadExportHandler, "get"),
		routerx.Post("/exports/preview", ipcontroller.PreviewExportHandler, "execute"),
		routerx.Get("/sync", ipcontroller.ScanSyncPoliciesHandler, "list"),
		routerx.Post("/sync", ipcontroller.CreateSyncPolicyHandler, "create"),
		routerx.Put("/sync/{id}", ipcontroller.UpdateSyncPolicyHandler, "update"),
		routerx.Delete("/sync/{id}", ipcontroller.DeleteSyncPolicyHandler, "delete"),
		routerx.Post("/sync/{id}/trigger", ipcontroller.TriggerSyncHandler, "execute"),
	)
}

func (m *Module) Start(ctx context.Context) error {
	ruleservice.RegisterIPDiscovery(runtimepkg.RegistryFromContext(ctx))
	m.service.StartSyncRunner(ctx)
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
