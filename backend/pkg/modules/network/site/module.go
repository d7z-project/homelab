package site

import (
	"context"
	controllerdeps "homelab/pkg/controllers"
	"homelab/pkg/controllers/middlewares"
	sitecontroller "homelab/pkg/controllers/network/site"
	runtimepkg "homelab/pkg/runtime"
	ipservice "homelab/pkg/services/network/ip"
	siteservice "homelab/pkg/services/network/site"
	ruleservice "homelab/pkg/services/rules"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	service  *siteservice.SitePoolService
	analysis *siteservice.AnalysisEngine
	exports  *siteservice.ExportManager
}

func New(enricher *ipservice.MMDBManager) *Module {
	analysis := siteservice.NewAnalysisEngine(enricher)
	exports := siteservice.NewExportManager(analysis)
	return &Module{
		service:  siteservice.NewSitePoolService(analysis, exports),
		analysis: analysis,
		exports:  exports,
	}
}

func (m *Module) Name() string { return "network.site" }

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/network/site", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AuditMiddleware("network/site"))
			r.Use(controllerdeps.WithSiteControllerDeps(m.service, m.analysis, m.exports))

			r.With(middlewares.RequirePermission("list", "network/site")).Get("/pools", sitecontroller.ScanSiteGroupsHandler)
			r.With(middlewares.RequirePermission("create", "network/site")).Post("/pools", sitecontroller.CreateSiteGroupHandler)
			r.With(middlewares.RequirePermission("update", "network/site")).Put("/pools/{id}", sitecontroller.UpdateSiteGroupHandler)
			r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/pools/{id}", sitecontroller.DeleteSiteGroupHandler)
			r.With(middlewares.RequirePermission("get", "network/site")).Get("/pools/{id}/preview", sitecontroller.PreviewSitePoolHandler)
			r.With(middlewares.RequirePermission("update", "network/site")).Post("/pools/{id}/entries", sitecontroller.ManageSitePoolEntryHandler)
			r.With(middlewares.RequirePermission("update", "network/site")).Delete("/pools/{id}/entries", sitecontroller.DeleteSitePoolEntryHandler)

			r.With(middlewares.RequirePermission("execute", "network/site")).Post("/analysis/hit-test", sitecontroller.SiteHitTestHandler)

			r.With(middlewares.RequirePermission("list", "network/site")).Get("/exports", sitecontroller.ScanSiteExportsHandler)
			r.With(middlewares.RequirePermission("list", "network/site")).Get("/exports/tasks", sitecontroller.ScanSiteExportTasksHandler)
			r.With(middlewares.RequirePermission("create", "network/site")).Post("/exports", sitecontroller.CreateSiteExportHandler)
			r.With(middlewares.RequirePermission("update", "network/site")).Put("/exports/{id}", sitecontroller.UpdateSiteExportHandler)
			r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/exports/{id}", sitecontroller.DeleteSiteExportHandler)
			r.With(middlewares.RequirePermission("execute", "network/site")).Post("/exports/{id}/trigger", sitecontroller.TriggerSiteExportHandler)
			r.With(middlewares.RequirePermission("get", "network/site")).Get("/exports/task/{taskId}", sitecontroller.SiteExportTaskStatusHandler)
			r.With(middlewares.RequirePermission("execute", "network/site")).Post("/exports/task/{taskId}/cancel", sitecontroller.CancelSiteExportTaskHandler)
			r.With(middlewares.RequirePermission("get", "network/site")).Get("/exports/download/{taskId}", sitecontroller.DownloadSiteExportHandler)
			r.With(middlewares.RequirePermission("execute", "network/site")).Post("/exports/preview", sitecontroller.PreviewSiteExportHandler)

			r.With(middlewares.RequirePermission("list", "network/site")).Get("/sync", sitecontroller.ScanSiteSyncPoliciesHandler)
			r.With(middlewares.RequirePermission("create", "network/site")).Post("/sync", sitecontroller.CreateSiteSyncPolicyHandler)
			r.With(middlewares.RequirePermission("update", "network/site")).Put("/sync/{id}", sitecontroller.UpdateSiteSyncPolicyHandler)
			r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/sync/{id}", sitecontroller.DeleteSiteSyncPolicyHandler)
			r.With(middlewares.RequirePermission("execute", "network/site")).Post("/sync/{id}/trigger", sitecontroller.TriggerSiteSyncHandler)
		})
	})
}

func (m *Module) Start(ctx context.Context) error {
	ruleservice.RegisterSiteDiscovery()
	siteservice.RegisterSiteProcessors(m.service)
	m.service.Start(ctx)
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
