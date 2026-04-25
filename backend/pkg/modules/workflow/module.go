package workflow

import (
	"context"

	"homelab/pkg/routes"
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
	routes.RegisterWorkflow(r)
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
