package workflow_test

import (
	"context"
	"strings"
	"testing"

	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	workflowmodel "homelab/pkg/models/workflow"
	moduleworkflow "homelab/pkg/modules/workflow"
	actionrepo "homelab/pkg/repositories/workflow/actions"
	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"

	"github.com/spf13/afero"
	"gopkg.d7z.net/middleware/kv"
	"gopkg.d7z.net/middleware/queue"
)

func TestModuleStartRegistersDiscoveryAndSAUsage(t *testing.T) {
	t.Parallel()

	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	registry := registryruntime.New()
	taskQueue := queue.NewMemoryQueue()
	deps := runtimepkg.ModuleDeps{
		Dependencies: runtimepkg.Dependencies{
			DB:     db,
			Queue:  taskQueue,
			FS:     afero.NewMemMapFs(),
			TempFS: afero.NewMemMapFs(),
		},
		Registry: registry,
	}
	ctx := deps.WithContext(context.Background())

	if err := actionrepo.SaveWorkflow(ctx, &workflowmodel.Workflow{
		ID: "wf-1",
		Meta: workflowmodel.WorkflowV1Meta{
			Name:             "deploy",
			Description:      "deploy workflow",
			Enabled:          true,
			ServiceAccountID: "sa-build",
			Vars:             map[string]workflowmodel.VarDefinition{},
			Steps: []workflowmodel.Step{
				{ID: "step1", Type: "core/sleep", Name: "Sleep", Params: map[string]string{"seconds": "1"}},
			},
		},
		Generation: 1,
	}); err != nil {
		t.Fatalf("seed workflow: %v", err)
	}

	module := moduleworkflow.New()
	if err := module.Init(deps); err != nil {
		t.Fatalf("init module: %v", err)
	}
	ctx = deps.WithContext(context.Background())
	if err := module.Start(ctx); err != nil {
		t.Fatalf("start module: %v", err)
	}

	ctx = commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{AllowedAll: true})
	lookup, err := registry.Lookup(ctx, discoverymodel.LookupRequest{
		Code:  "actions/workflows",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("lookup after module start: %v", err)
	}
	if len(lookup.Items) != 1 {
		t.Fatalf("expected 1 workflow item, got %d", len(lookup.Items))
	}

	err = registry.CheckSAUsage(ctx, "sa-build")
	if err == nil || !strings.Contains(err.Error(), "deploy") {
		t.Fatalf("expected SA usage error mentioning workflow, got %v", err)
	}
}
