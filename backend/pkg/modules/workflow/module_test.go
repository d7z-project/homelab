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
	"homelab/pkg/testkit"
)

func TestModuleStartRegistersDiscoveryAndSAUsage(t *testing.T) {
	t.Parallel()

	env := testkit.StartApp(t,
		testkit.SeedModule("test.seed.workflow", func(ctx context.Context, _ runtimepkg.ModuleDeps) error {
			return actionrepo.SaveWorkflow(ctx, &workflowmodel.Workflow{
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
			})
		}),
		moduleworkflow.New(),
	)

	ctx := commonauth.WithPermissions(env.Context(), &rbacmodel.ResourcePermissions{AllowedAll: true})
	lookup, err := env.Deps.Registry.Lookup(ctx, discoverymodel.LookupRequest{
		Code:  "actions/workflows",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("lookup after module start: %v", err)
	}
	if len(lookup.Items) != 1 {
		t.Fatalf("expected 1 workflow item, got %d", len(lookup.Items))
	}

	err = env.Deps.Registry.CheckSAUsage(ctx, "sa-build")
	if err == nil || !strings.Contains(err.Error(), "deploy") {
		t.Fatalf("expected SA usage error mentioning workflow, got %v", err)
	}
}
