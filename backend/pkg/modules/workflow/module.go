package workflow

import (
	"context"

	"homelab/pkg/controllers/routerx"
	workflowcontroller "homelab/pkg/controllers/workflow"
	workflowrepo "homelab/pkg/repositories/workflow/actions"
	runtimepkg "homelab/pkg/runtime"
	actionservice "homelab/pkg/services/workflow"
	actionprocessors "homelab/pkg/services/workflow/processors"
)

type Module struct {
	runtime *actionservice.Runtime
}

func New() *Module {
	return &Module{}
}

func (m *Module) Name() string {
	return "workflow"
}

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	workflowrepo.Configure(deps.DB)
	rt, err := actionservice.NewRuntime(deps)
	if err != nil {
		return err
	}
	m.runtime = rt
	return nil
}

func (m *Module) Routes() runtimepkg.RouteHandler {
	return routerx.New("/actions",
		routerx.Use(actionservice.ContextMiddleware(m.runtime)),
		routerx.Group("",
			routerx.Routes(
				routerx.Get("/webhooks/{token}", workflowcontroller.WebhookHandler),
				routerx.Post("/webhooks/{token}", workflowcontroller.WebhookHandler),
			),
		),
		routerx.Group("",
			routerx.WithScope(routerx.Scope{
				Resource: "actions",
				Audit:    "actions",
				UsesAuth: true,
			}),
			routerx.Routes(
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
			),
		),
	)
}

func (m *Module) Start(ctx context.Context) error {
	actionprocessors.RegisterBuiltins()
	actionservice.RegisterDiscovery(m.runtime.Deps.Registry)
	actionservice.BootUpSelfHealing(m.runtime.WithContext(ctx))
	if err := m.runtime.TriggerManager.InitTriggers(ctx); err != nil {
		return err
	}
	if err := m.runtime.StartExecutionConsumer(ctx); err != nil {
		return err
	}
	m.runtime.TriggerManager.Start(ctx)
	return nil
}

func (m *Module) Stop(ctx context.Context) error {
	_ = ctx
	return nil
}

var _ runtimepkg.Module = (*Module)(nil)
