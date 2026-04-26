package workflow_test

import (
	"context"
	"strings"
	"testing"

	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	workflowmodel "homelab/pkg/models/workflow"
	actionrepo "homelab/pkg/repositories/workflow/actions"
	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"
	actionservice "homelab/pkg/services/workflow"

	"github.com/spf13/afero"
	"gopkg.d7z.net/middleware/kv"
)

func TestRegisterDiscovery(t *testing.T) {
	t.Parallel()

	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	registry := registryruntime.New()
	deps := runtimepkg.ModuleDeps{
		Dependencies: runtimepkg.Dependencies{
			DB:     db,
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

	actionservice.RegisterDiscovery(registry)

	ctx = commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{AllowedAll: true})

	lookup, err := registry.Lookup(ctx, discoverymodel.LookupRequest{
		Code:  "actions/workflows",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("lookup workflows: %v", err)
	}
	if len(lookup.Items) != 1 || lookup.Items[0].Name != "deploy" {
		t.Fatalf("unexpected workflow lookup result: %#v", lookup.Items)
	}

	suggestions, err := registry.SuggestResources(ctx, "actions/workflows/")
	if err != nil {
		t.Fatalf("suggest resources: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("expected actions resource suggestions")
	}

	err = registry.CheckSAUsage(ctx, "sa-build")
	if err == nil || !strings.Contains(err.Error(), "deploy") {
		t.Fatalf("expected SA usage error mentioning workflow, got %v", err)
	}
}
