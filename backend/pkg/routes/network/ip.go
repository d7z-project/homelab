package network

import (
	controllerdeps "homelab/pkg/controllers"
	"homelab/pkg/controllers/middlewares"
	ipcontroller "homelab/pkg/controllers/network/ip"
	"homelab/pkg/services/network/ip"

	"github.com/go-chi/chi/v5"
)

func RegisterIP(r chi.Router, poolService *ip.IPPoolService, analysis *ip.AnalysisEngine, exports *ip.ExportManager) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.AuditMiddleware("network/ip"))
		r.Use(controllerdeps.WithIPControllerDeps(poolService, analysis, exports))

		r.With(middlewares.RequirePermission("list", "network/ip")).Get("/api/v1/network/ip/pools", ipcontroller.ScanGroupsHandler)
		r.With(middlewares.RequirePermission("create", "network/ip")).Post("/api/v1/network/ip/pools", ipcontroller.CreateGroupHandler)
		r.With(middlewares.RequirePermission("update", "network/ip")).Put("/api/v1/network/ip/pools/{id}", ipcontroller.UpdateGroupHandler)
		r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/api/v1/network/ip/pools/{id}", ipcontroller.DeleteGroupHandler)
		r.With(middlewares.RequirePermission("get", "network/ip")).Get("/api/v1/network/ip/pools/{id}/preview", ipcontroller.PreviewPoolHandler)
		r.With(middlewares.RequirePermission("update", "network/ip")).Post("/api/v1/network/ip/pools/{id}/entries", ipcontroller.ManagePoolEntryHandler)
		r.With(middlewares.RequirePermission("update", "network/ip")).Delete("/api/v1/network/ip/pools/{id}/entries", ipcontroller.DeletePoolEntryHandler)

		r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/api/v1/network/ip/analysis/hit-test", ipcontroller.HitTestHandler)
		r.With(middlewares.RequirePermission("get", "network/ip")).Get("/api/v1/network/ip/analysis/info", ipcontroller.IPInfoHandler)

		r.With(middlewares.RequirePermission("list", "network/ip")).Get("/api/v1/network/ip/exports", ipcontroller.ScanExportsHandler)
		r.With(middlewares.RequirePermission("list", "network/ip")).Get("/api/v1/network/ip/exports/tasks", ipcontroller.ScanExportTasksHandler)
		r.With(middlewares.RequirePermission("create", "network/ip")).Post("/api/v1/network/ip/exports", ipcontroller.CreateExportHandler)
		r.With(middlewares.RequirePermission("update", "network/ip")).Put("/api/v1/network/ip/exports/{id}", ipcontroller.UpdateExportHandler)
		r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/api/v1/network/ip/exports/{id}", ipcontroller.DeleteExportHandler)
		r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/api/v1/network/ip/exports/{id}/trigger", ipcontroller.TriggerExportHandler)
		r.With(middlewares.RequirePermission("get", "network/ip")).Get("/api/v1/network/ip/exports/task/{taskId}", ipcontroller.ExportTaskStatusHandler)
		r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/api/v1/network/ip/exports/task/{taskId}/cancel", ipcontroller.CancelExportTaskHandler)
		r.With(middlewares.RequirePermission("get", "network/ip")).Get("/api/v1/network/ip/exports/download/{taskId}", ipcontroller.DownloadExportHandler)
		r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/api/v1/network/ip/exports/preview", ipcontroller.PreviewExportHandler)

		r.With(middlewares.RequirePermission("list", "network/ip")).Get("/api/v1/network/ip/sync", ipcontroller.ScanSyncPoliciesHandler)
		r.With(middlewares.RequirePermission("create", "network/ip")).Post("/api/v1/network/ip/sync", ipcontroller.CreateSyncPolicyHandler)
		r.With(middlewares.RequirePermission("update", "network/ip")).Put("/api/v1/network/ip/sync/{id}", ipcontroller.UpdateSyncPolicyHandler)
		r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/api/v1/network/ip/sync/{id}", ipcontroller.DeleteSyncPolicyHandler)
		r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/api/v1/network/ip/sync/{id}/trigger", ipcontroller.TriggerSyncHandler)
	})
}
