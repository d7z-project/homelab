package workflow

import (
	"homelab/pkg/controllers/middlewares"
	workflowcontroller "homelab/pkg/controllers/workflow"

	"github.com/go-chi/chi/v5"
)

func Register(r chi.Router) {
	r.Get("/api/v1/actions/webhooks/{token}", workflowcontroller.WebhookHandler)
	r.Post("/api/v1/actions/webhooks/{token}", workflowcontroller.WebhookHandler)

	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.AuditMiddleware("actions"))

		r.With(middlewares.RequirePermission("list", "actions")).Get("/api/v1/actions/workflows", workflowcontroller.ScanWorkflowsHandler)
		r.With(middlewares.RequirePermission("create", "actions")).Post("/api/v1/actions/workflows", workflowcontroller.CreateWorkflowHandler)
		r.With(middlewares.RequirePermission("get", "actions")).Get("/api/v1/actions/workflows/schema", workflowcontroller.GetWorkflowSchemaHandler)
		r.With(middlewares.RequirePermission("execute", "actions")).Post("/api/v1/actions/workflows/validate", workflowcontroller.ValidateWorkflowHandler)
		r.With(middlewares.RequirePermission("execute", "actions")).Post("/api/v1/actions/validate/regex", workflowcontroller.ValidateRegexHandler)
		r.With(middlewares.RequirePermission("update", "actions")).Put("/api/v1/actions/workflows/{id}", workflowcontroller.UpdateWorkflowHandler)
		r.With(middlewares.RequirePermission("get", "actions")).Get("/api/v1/actions/workflows/{id}", workflowcontroller.GetWorkflowHandler)
		r.With(middlewares.RequirePermission("delete", "actions")).Delete("/api/v1/actions/workflows/{id}", workflowcontroller.DeleteWorkflowHandler)
		r.With(middlewares.RequirePermission("execute", "actions")).Post("/api/v1/actions/workflows/{workflowId}/run", workflowcontroller.RunWorkflowHandler)
		r.With(middlewares.RequirePermission("update", "actions")).Post("/api/v1/actions/workflows/{id}/webhook/reset", workflowcontroller.ResetWebhookTokenHandler)

		r.With(middlewares.RequirePermission("list", "actions")).Get("/api/v1/actions/instances", workflowcontroller.ScanInstancesHandler)
		r.With(middlewares.RequirePermission("get", "actions")).Get("/api/v1/actions/instances/{id}", workflowcontroller.GetInstanceHandler)
		r.With(middlewares.RequirePermission("delete", "actions")).Post("/api/v1/actions/instances/cleanup", workflowcontroller.CleanupInstancesHandler)
		r.With(middlewares.RequirePermission("get", "actions")).Get("/api/v1/actions/instances/{id}/logs", workflowcontroller.GetInstanceLogsHandler)
		r.With(middlewares.RequirePermission("delete", "actions")).Delete("/api/v1/actions/instances/{id}", workflowcontroller.DeleteInstanceHandler)
		r.With(middlewares.RequirePermission("execute", "actions")).Post("/api/v1/actions/instances/{id}/cancel", workflowcontroller.CancelInstanceHandler)

		r.With(middlewares.RequirePermission("list", "actions")).Get("/api/v1/actions/manifests", workflowcontroller.ScanManifestsHandler)
		r.With(middlewares.RequirePermission("execute", "actions")).Post("/api/v1/actions/probe", workflowcontroller.ProbeHandler)
	})
}
