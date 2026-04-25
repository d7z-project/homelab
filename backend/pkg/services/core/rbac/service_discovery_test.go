package rbac_test

import (
	"context"
	"testing"

	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	registryruntime "homelab/pkg/runtime/registry"
	rbacservice "homelab/pkg/services/core/rbac"

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

	if err := rbacrepo.ServiceAccountRepo.Cow(context.Background(), "sa-1", func(res *rbacmodel.ServiceAccount) error {
		res.ID = "sa-1"
		res.Meta = rbacmodel.ServiceAccountV1Meta{Name: "builder", Comments: "build bot", Enabled: true}
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	}); err != nil {
		t.Fatalf("seed service account: %v", err)
	}
	if err := rbacrepo.RoleRepo.Cow(context.Background(), "role-1", func(res *rbacmodel.Role) error {
		res.ID = "role-1"
		res.Meta = rbacmodel.RoleV1Meta{Name: "admin", Comments: "admin role", Rules: []rbacmodel.PolicyRule{{Resource: "rbac", Verbs: []string{"*"}}}}
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	}); err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := rbacrepo.BindingRepo.Cow(context.Background(), "binding-1", func(res *rbacmodel.RoleBinding) error {
		res.ID = "binding-1"
		res.Meta = rbacmodel.RoleBindingV1Meta{Name: "binding-1", RoleIDs: []string{"role-1"}, ServiceAccountID: "sa-1", Enabled: true}
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	}); err != nil {
		t.Fatalf("seed role binding: %v", err)
	}

	rbacservice.RegisterDiscovery()

	ctx := commonauth.WithPermissions(context.Background(), &rbacmodel.ResourcePermissions{AllowedAll: true})

	for _, code := range []string{"rbac/serviceaccounts", "rbac/roles", "rbac/rolebindings"} {
		res, err := registryruntime.Default().Lookup(ctx, discoverymodel.LookupRequest{Code: code, Limit: 20})
		if err != nil {
			t.Fatalf("lookup %s: %v", code, err)
		}
		if len(res.Items) != 1 {
			t.Fatalf("unexpected lookup size for %s: %#v", code, res.Items)
		}
	}

	suggestions, err := registryruntime.Default().SuggestResources(ctx, "rbac/")
	if err != nil {
		t.Fatalf("suggest resources: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("expected rbac resource suggestions")
	}
}
