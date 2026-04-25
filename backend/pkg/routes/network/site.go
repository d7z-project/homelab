package network

import (
	controllerdeps "homelab/pkg/controllers"
	"homelab/pkg/controllers/middlewares"
	sitecontroller "homelab/pkg/controllers/network/site"
	"homelab/pkg/services/network/site"

	"github.com/go-chi/chi/v5"
)

func RegisterSite(r chi.Router, poolService *site.SitePoolService, analysis *site.AnalysisEngine, exports *site.ExportManager) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.AuditMiddleware("network/site"))
		r.Use(controllerdeps.WithSiteControllerDeps(poolService, analysis, exports))

		r.With(middlewares.RequirePermission("list", "network/site")).Get("/api/v1/network/site/pools", sitecontroller.ScanSiteGroupsHandler)
		r.With(middlewares.RequirePermission("create", "network/site")).Post("/api/v1/network/site/pools", sitecontroller.CreateSiteGroupHandler)
		r.With(middlewares.RequirePermission("update", "network/site")).Put("/api/v1/network/site/pools/{id}", sitecontroller.UpdateSiteGroupHandler)
		r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/api/v1/network/site/pools/{id}", sitecontroller.DeleteSiteGroupHandler)
		r.With(middlewares.RequirePermission("get", "network/site")).Get("/api/v1/network/site/pools/{id}/preview", sitecontroller.PreviewSitePoolHandler)
		r.With(middlewares.RequirePermission("update", "network/site")).Post("/api/v1/network/site/pools/{id}/entries", sitecontroller.ManageSitePoolEntryHandler)
		r.With(middlewares.RequirePermission("update", "network/site")).Delete("/api/v1/network/site/pools/{id}/entries", sitecontroller.DeleteSitePoolEntryHandler)

		r.With(middlewares.RequirePermission("execute", "network/site")).Post("/api/v1/network/site/analysis/hit-test", sitecontroller.SiteHitTestHandler)

		r.With(middlewares.RequirePermission("list", "network/site")).Get("/api/v1/network/site/exports", sitecontroller.ScanSiteExportsHandler)
		r.With(middlewares.RequirePermission("list", "network/site")).Get("/api/v1/network/site/exports/tasks", sitecontroller.ScanSiteExportTasksHandler)
		r.With(middlewares.RequirePermission("create", "network/site")).Post("/api/v1/network/site/exports", sitecontroller.CreateSiteExportHandler)
		r.With(middlewares.RequirePermission("update", "network/site")).Put("/api/v1/network/site/exports/{id}", sitecontroller.UpdateSiteExportHandler)
		r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/api/v1/network/site/exports/{id}", sitecontroller.DeleteSiteExportHandler)
		r.With(middlewares.RequirePermission("execute", "network/site")).Post("/api/v1/network/site/exports/{id}/trigger", sitecontroller.TriggerSiteExportHandler)
		r.With(middlewares.RequirePermission("get", "network/site")).Get("/api/v1/network/site/exports/task/{taskId}", sitecontroller.SiteExportTaskStatusHandler)
		r.With(middlewares.RequirePermission("execute", "network/site")).Post("/api/v1/network/site/exports/task/{taskId}/cancel", sitecontroller.CancelSiteExportTaskHandler)
		r.With(middlewares.RequirePermission("get", "network/site")).Get("/api/v1/network/site/exports/download/{taskId}", sitecontroller.DownloadSiteExportHandler)
		r.With(middlewares.RequirePermission("execute", "network/site")).Post("/api/v1/network/site/exports/preview", sitecontroller.PreviewSiteExportHandler)

		r.With(middlewares.RequirePermission("list", "network/site")).Get("/api/v1/network/site/sync", sitecontroller.ScanSiteSyncPoliciesHandler)
		r.With(middlewares.RequirePermission("create", "network/site")).Post("/api/v1/network/site/sync", sitecontroller.CreateSiteSyncPolicyHandler)
		r.With(middlewares.RequirePermission("update", "network/site")).Put("/api/v1/network/site/sync/{id}", sitecontroller.UpdateSiteSyncPolicyHandler)
		r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/api/v1/network/site/sync/{id}", sitecontroller.DeleteSiteSyncPolicyHandler)
		r.With(middlewares.RequirePermission("execute", "network/site")).Post("/api/v1/network/site/sync/{id}/trigger", sitecontroller.TriggerSiteSyncHandler)
	})
}
