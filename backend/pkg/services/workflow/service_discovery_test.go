package workflow_test

import (
	"strings"
	"testing"

	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	workflowmodel "homelab/pkg/models/workflow"
	actionrepo "homelab/pkg/repositories/workflow/actions"
	actionservice "homelab/pkg/services/workflow"
	"homelab/pkg/testkit"
)

func TestRegisterDiscovery(t *testing.T) {
	t.Parallel()

	deps := testkit.NewModuleDeps(t)
	ctx := deps.WithContext(t.Context())

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

	actionservice.RegisterDiscovery(deps.Registry)

	ctx = commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{AllowedAll: true})

	lookup, err := deps.Registry.Lookup(ctx, discoverymodel.LookupRequest{
		Code:  "actions/workflows",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("lookup workflows: %v", err)
	}
	if len(lookup.Items) != 1 || lookup.Items[0].Name != "deploy" {
		t.Fatalf("unexpected workflow lookup result: %#v", lookup.Items)
	}

	suggestions, err := deps.Registry.SuggestResources(ctx, "actions/workflows/")
	if err != nil {
		t.Fatalf("suggest resources: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("expected actions resource suggestions")
	}

	err = deps.Registry.CheckSAUsage(ctx, "sa-build")
	if err == nil || !strings.Contains(err.Error(), "deploy") {
		t.Fatalf("expected SA usage error mentioning workflow, got %v", err)
	}
}
