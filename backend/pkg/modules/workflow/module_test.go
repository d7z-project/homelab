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
	moduleworkflow "homelab/pkg/modules/workflow"
	actionrepo "homelab/pkg/repositories/workflow/actions"
	registryruntime "homelab/pkg/runtime/registry"

	"github.com/spf13/afero"
	"gopkg.d7z.net/middleware/kv"
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
	common.DB = db
	common.FS = afero.NewMemMapFs()
	common.TempDir = afero.NewMemMapFs()

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

	module := moduleworkflow.New()
	if err := module.Start(context.Background()); err != nil {
		t.Fatalf("start module: %v", err)
	}

	ctx := commonauth.WithPermissions(context.Background(), &rbacmodel.ResourcePermissions{AllowedAll: true})
	lookup, err := registryruntime.Default().Lookup(ctx, discoverymodel.LookupRequest{
		Code:  "actions/workflows",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("lookup after module start: %v", err)
	}
	if len(lookup.Items) != 1 {
		t.Fatalf("expected 1 workflow item, got %d", len(lookup.Items))
	}

	err = registryruntime.Default().CheckSAUsage(ctx, "sa-build")
	if err == nil || !strings.Contains(err.Error(), "deploy") {
		t.Fatalf("expected SA usage error mentioning workflow, got %v", err)
	}
}
