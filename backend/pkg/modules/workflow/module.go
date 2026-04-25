package workflow

import (
	"context"

	"homelab/pkg/controllers/middlewares"
	workflowcontroller "homelab/pkg/controllers/workflow"
	runtimepkg "homelab/pkg/runtime"
	actionservice "homelab/pkg/services/workflow"
	actionprocessors "homelab/pkg/services/workflow/processors"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "workflow"
}

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/actions", func(r chi.Router) {
		r.Get("/webhooks/{token}", workflowcontroller.WebhookHandler)
		r.Post("/webhooks/{token}", workflowcontroller.WebhookHandler)

		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AuditMiddleware("actions"))

			r.With(middlewares.RequirePermission("list", "actions")).Get("/workflows", workflowcontroller.ScanWorkflowsHandler)
			r.With(middlewares.RequirePermission("create", "actions")).Post("/workflows", workflowcontroller.CreateWorkflowHandler)
			r.With(middlewares.RequirePermission("get", "actions")).Get("/workflows/schema", workflowcontroller.GetWorkflowSchemaHandler)
			r.With(middlewares.RequirePermission("execute", "actions")).Post("/workflows/validate", workflowcontroller.ValidateWorkflowHandler)
			r.With(middlewares.RequirePermission("execute", "actions")).Post("/validate/regex", workflowcontroller.ValidateRegexHandler)
			r.With(middlewares.RequirePermission("update", "actions")).Put("/workflows/{id}", workflowcontroller.UpdateWorkflowHandler)
			r.With(middlewares.RequirePermission("get", "actions")).Get("/workflows/{id}", workflowcontroller.GetWorkflowHandler)
			r.With(middlewares.RequirePermission("delete", "actions")).Delete("/workflows/{id}", workflowcontroller.DeleteWorkflowHandler)
			r.With(middlewares.RequirePermission("execute", "actions")).Post("/workflows/{workflowId}/run", workflowcontroller.RunWorkflowHandler)
			r.With(middlewares.RequirePermission("update", "actions")).Post("/workflows/{id}/webhook/reset", workflowcontroller.ResetWebhookTokenHandler)

			r.With(middlewares.RequirePermission("list", "actions")).Get("/instances", workflowcontroller.ScanInstancesHandler)
			r.With(middlewares.RequirePermission("get", "actions")).Get("/instances/{id}", workflowcontroller.GetInstanceHandler)
			r.With(middlewares.RequirePermission("delete", "actions")).Post("/instances/cleanup", workflowcontroller.CleanupInstancesHandler)
			r.With(middlewares.RequirePermission("get", "actions")).Get("/instances/{id}/logs", workflowcontroller.GetInstanceLogsHandler)
			r.With(middlewares.RequirePermission("delete", "actions")).Delete("/instances/{id}", workflowcontroller.DeleteInstanceHandler)
			r.With(middlewares.RequirePermission("execute", "actions")).Post("/instances/{id}/cancel", workflowcontroller.CancelInstanceHandler)

			r.With(middlewares.RequirePermission("list", "actions")).Get("/manifests", workflowcontroller.ScanManifestsHandler)
			r.With(middlewares.RequirePermission("execute", "actions")).Post("/probe", workflowcontroller.ProbeHandler)
		})
	})
}

func (m *Module) Start(ctx context.Context) error {
	actionprocessors.RegisterBuiltins()
	actionservice.Init()
	actionservice.RegisterDiscovery()
	actionservice.BootUpSelfHealing()
	if err := actionservice.GlobalTriggerManager.InitTriggers(ctx); err != nil {
		return err
	}
	actionservice.GlobalTriggerManager.Start()
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	_ = ctx
	return nil
}

var _ runtimepkg.Module = (*Module)(nil)
