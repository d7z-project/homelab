package workflow_test

import (
	"context"
	"strings"
	"testing"

	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	workflowmodel "homelab/pkg/models/workflow"
	actionrepo "homelab/pkg/repositories/workflow/actions"
	registryruntime "homelab/pkg/runtime/registry"
	actionservice "homelab/pkg/services/workflow"

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
	common.DB = db

	if err := actionrepo.WorkflowRepo.Cow(context.Background(), "wf-1", func(res *workflowmodel.Workflow) error {
		res.ID = "wf-1"
		res.Meta = workflowmodel.WorkflowV1Meta{
			Name:             "deploy",
			Description:      "deploy workflow",
			Enabled:          true,
			ServiceAccountID: "sa-build",
			Vars:             map[string]workflowmodel.VarDefinition{},
			Steps: []workflowmodel.Step{
				{ID: "step1", Type: "core/sleep", Name: "Sleep", Params: map[string]string{"seconds": "1"}},
			},
		}
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	}); err != nil {
		t.Fatalf("seed workflow: %v", err)
	}

	actionservice.RegisterDiscovery()

	ctx := commonauth.WithPermissions(context.Background(), &rbacmodel.ResourcePermissions{AllowedAll: true})

	lookup, err := registryruntime.Default().Lookup(ctx, discoverymodel.LookupRequest{
		Code:  "actions/workflows",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("lookup workflows: %v", err)
	}
	if len(lookup.Items) != 1 || lookup.Items[0].Name != "deploy" {
		t.Fatalf("unexpected workflow lookup result: %#v", lookup.Items)
	}

	suggestions, err := registryruntime.Default().SuggestResources(ctx, "actions/workflows/")
	if err != nil {
		t.Fatalf("suggest resources: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("expected actions resource suggestions")
	}

	err = registryruntime.Default().CheckSAUsage(ctx, "sa-build")
	if err == nil || !strings.Contains(err.Error(), "deploy") {
		t.Fatalf("expected SA usage error mentioning workflow, got %v", err)
	}
}
