package ip

import (
	"context"
	controllerdeps "homelab/pkg/controllers"
	"homelab/pkg/controllers/middlewares"
	ipcontroller "homelab/pkg/controllers/network/ip"
	runtimepkg "homelab/pkg/runtime"
	ipservice "homelab/pkg/services/network/ip"
	ruleservice "homelab/pkg/services/rules"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	service  *ipservice.IPPoolService
	analysis *ipservice.AnalysisEngine
	exports  *ipservice.ExportManager
}

func New(enricher *ipservice.MMDBManager) *Module {
	analysis := ipservice.NewAnalysisEngine(enricher)
	exports := ipservice.NewExportManager(analysis)
	return &Module{
		service:  ipservice.NewIPPoolService(analysis, exports),
		analysis: analysis,
		exports:  exports,
	}
}

func (m *Module) Name() string { return "network.ip" }

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/network/ip", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AuditMiddleware("network/ip"))
			r.Use(controllerdeps.WithIPControllerDeps(m.service, m.analysis, m.exports))

			r.With(middlewares.RequirePermission("list", "network/ip")).Get("/pools", ipcontroller.ScanGroupsHandler)
			r.With(middlewares.RequirePermission("create", "network/ip")).Post("/pools", ipcontroller.CreateGroupHandler)
			r.With(middlewares.RequirePermission("update", "network/ip")).Put("/pools/{id}", ipcontroller.UpdateGroupHandler)
			r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/pools/{id}", ipcontroller.DeleteGroupHandler)
			r.With(middlewares.RequirePermission("get", "network/ip")).Get("/pools/{id}/preview", ipcontroller.PreviewPoolHandler)
			r.With(middlewares.RequirePermission("update", "network/ip")).Post("/pools/{id}/entries", ipcontroller.ManagePoolEntryHandler)
			r.With(middlewares.RequirePermission("update", "network/ip")).Delete("/pools/{id}/entries", ipcontroller.DeletePoolEntryHandler)

			r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/analysis/hit-test", ipcontroller.HitTestHandler)
			r.With(middlewares.RequirePermission("get", "network/ip")).Get("/analysis/info", ipcontroller.IPInfoHandler)

			r.With(middlewares.RequirePermission("list", "network/ip")).Get("/exports", ipcontroller.ScanExportsHandler)
			r.With(middlewares.RequirePermission("list", "network/ip")).Get("/exports/tasks", ipcontroller.ScanExportTasksHandler)
			r.With(middlewares.RequirePermission("create", "network/ip")).Post("/exports", ipcontroller.CreateExportHandler)
			r.With(middlewares.RequirePermission("update", "network/ip")).Put("/exports/{id}", ipcontroller.UpdateExportHandler)
			r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/exports/{id}", ipcontroller.DeleteExportHandler)
			r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/exports/{id}/trigger", ipcontroller.TriggerExportHandler)
			r.With(middlewares.RequirePermission("get", "network/ip")).Get("/exports/task/{taskId}", ipcontroller.ExportTaskStatusHandler)
			r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/exports/task/{taskId}/cancel", ipcontroller.CancelExportTaskHandler)
			r.With(middlewares.RequirePermission("get", "network/ip")).Get("/exports/download/{taskId}", ipcontroller.DownloadExportHandler)
			r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/exports/preview", ipcontroller.PreviewExportHandler)

			r.With(middlewares.RequirePermission("list", "network/ip")).Get("/sync", ipcontroller.ScanSyncPoliciesHandler)
			r.With(middlewares.RequirePermission("create", "network/ip")).Post("/sync", ipcontroller.CreateSyncPolicyHandler)
			r.With(middlewares.RequirePermission("update", "network/ip")).Put("/sync/{id}", ipcontroller.UpdateSyncPolicyHandler)
			r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/sync/{id}", ipcontroller.DeleteSyncPolicyHandler)
			r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/sync/{id}/trigger", ipcontroller.TriggerSyncHandler)
		})
	})
}

func (m *Module) Start(ctx context.Context) error {
	ruleservice.RegisterIPDiscovery()
	m.service.StartSyncRunner(ctx)
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
