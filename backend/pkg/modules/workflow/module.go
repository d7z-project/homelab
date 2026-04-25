package workflow

import (
	"context"

	"homelab/pkg/controllers/routerx"
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

		routerx.Mount(r, "/", routerx.Scope{
			Resource: "actions",
			Audit:    "actions",
			UsesAuth: true,
		},
			routerx.Get("/workflows", workflowcontroller.ScanWorkflowsHandler, "list"),
			routerx.Post("/workflows", workflowcontroller.CreateWorkflowHandler, "create"),
			routerx.Get("/workflows/schema", workflowcontroller.GetWorkflowSchemaHandler, "get"),
			routerx.Post("/workflows/validate", workflowcontroller.ValidateWorkflowHandler, "execute"),
			routerx.Post("/validate/regex", workflowcontroller.ValidateRegexHandler, "execute"),
			routerx.Put("/workflows/{id}", workflowcontroller.UpdateWorkflowHandler, "update"),
			routerx.Get("/workflows/{id}", workflowcontroller.GetWorkflowHandler, "get"),
			routerx.Delete("/workflows/{id}", workflowcontroller.DeleteWorkflowHandler, "delete"),
			routerx.Post("/workflows/{workflowId}/run", workflowcontroller.RunWorkflowHandler, "execute"),
			routerx.Post("/workflows/{id}/webhook/reset", workflowcontroller.ResetWebhookTokenHandler, "update"),
			routerx.Get("/instances", workflowcontroller.ScanInstancesHandler, "list"),
			routerx.Get("/instances/{id}", workflowcontroller.GetInstanceHandler, "get"),
			routerx.Post("/instances/cleanup", workflowcontroller.CleanupInstancesHandler, "delete"),
			routerx.Get("/instances/{id}/logs", workflowcontroller.GetInstanceLogsHandler, "get"),
			routerx.Delete("/instances/{id}", workflowcontroller.DeleteInstanceHandler, "delete"),
			routerx.Post("/instances/{id}/cancel", workflowcontroller.CancelInstanceHandler, "execute"),
			routerx.Get("/manifests", workflowcontroller.ScanManifestsHandler, "list"),
			routerx.Post("/probe", workflowcontroller.ProbeHandler, "execute"),
		)
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
