package site

import (
	"context"
	"net/http"

	sitecontroller "homelab/pkg/controllers/network/site"
	"homelab/pkg/controllers/routerx"
	siterepo "homelab/pkg/repositories/network/site"
	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"
	ipservice "homelab/pkg/services/network/ip"
	siteservice "homelab/pkg/services/network/site"
	ruleservice "homelab/pkg/services/rules"

	"github.com/spf13/afero"
)

type Module struct {
	enricher *ipservice.MMDBManager
	service  *siteservice.SitePoolService
	analysis *siteservice.AnalysisEngine
	exports  *siteservice.ExportManager
	tempFS   http.FileSystem
	registry *registryruntime.Registry
}

func New(enricher *ipservice.MMDBManager) *Module {
	return &Module{enricher: enricher}
}

func (m *Module) Name() string { return "network.site" }

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	siterepo.Configure(deps.DB)
	m.analysis = siteservice.NewAnalysisEngine(deps, m.enricher)
	m.exports = siteservice.NewExportManager(deps, m.analysis)
	m.service = siteservice.NewSitePoolService(deps, m.analysis, m.exports)
	m.tempFS = afero.NewHttpFs(deps.TempFS)
	m.registry = deps.Registry
	return nil
}

func (m *Module) Routes() runtimepkg.RouteHandler {
	return routerx.New("/network/site",
		routerx.WithScope(routerx.Scope{
			Resource: "network/site",
			Audit:    "network/site",
			UsesAuth: true,
			Extra: []func(http.Handler) http.Handler{
				sitecontroller.WithControllerDeps(m.service, m.analysis, m.exports, m.tempFS),
			},
		}),
		routerx.Routes(
			routerx.Get("/pools", sitecontroller.ScanSiteGroupsHandler, "list"),
			routerx.Post("/pools", sitecontroller.CreateSiteGroupHandler, "create"),
			routerx.Put("/pools/{id}", sitecontroller.UpdateSiteGroupHandler, "update"),
			routerx.Delete("/pools/{id}", sitecontroller.DeleteSiteGroupHandler, "delete"),
			routerx.Get("/pools/{id}/preview", sitecontroller.PreviewSitePoolHandler, "get"),
			routerx.Post("/pools/{id}/entries", sitecontroller.ManageSitePoolEntryHandler, "update"),
			routerx.Delete("/pools/{id}/entries", sitecontroller.DeleteSitePoolEntryHandler, "update"),
			routerx.Post("/analysis/hit-test", sitecontroller.SiteHitTestHandler, "execute"),
			routerx.Get("/exports", sitecontroller.ScanSiteExportsHandler, "list"),
			routerx.Get("/exports/tasks", sitecontroller.ScanSiteExportTasksHandler, "list"),
			routerx.Post("/exports", sitecontroller.CreateSiteExportHandler, "create"),
			routerx.Put("/exports/{id}", sitecontroller.UpdateSiteExportHandler, "update"),
			routerx.Delete("/exports/{id}", sitecontroller.DeleteSiteExportHandler, "delete"),
			routerx.Post("/exports/{id}/trigger", sitecontroller.TriggerSiteExportHandler, "execute"),
			routerx.Get("/exports/task/{taskId}", sitecontroller.SiteExportTaskStatusHandler, "get"),
			routerx.Post("/exports/task/{taskId}/cancel", sitecontroller.CancelSiteExportTaskHandler, "execute"),
			routerx.Get("/exports/download/{taskId}", sitecontroller.DownloadSiteExportHandler, "get"),
			routerx.Post("/exports/preview", sitecontroller.PreviewSiteExportHandler, "execute"),
			routerx.Get("/sync", sitecontroller.ScanSiteSyncPoliciesHandler, "list"),
			routerx.Post("/sync", sitecontroller.CreateSiteSyncPolicyHandler, "create"),
			routerx.Put("/sync/{id}", sitecontroller.UpdateSiteSyncPolicyHandler, "update"),
			routerx.Delete("/sync/{id}", sitecontroller.DeleteSiteSyncPolicyHandler, "delete"),
			routerx.Post("/sync/{id}/trigger", sitecontroller.TriggerSiteSyncHandler, "execute"),
		),
	)
}

func (m *Module) Start(ctx context.Context) error {
	if m.enricher != nil {
		if err := m.enricher.Init(ctx); err != nil {
			return err
		}
	}
	ruleservice.RegisterSiteDiscovery(m.registry)
	siteservice.RegisterSiteProcessors(m.service)
	return m.service.Start(ctx)
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
