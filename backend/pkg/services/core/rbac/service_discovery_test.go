package rbac_test

import (
	"context"
	"testing"

	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"
	rbacservice "homelab/pkg/services/core/rbac"

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

	if err := rbacrepo.SaveServiceAccount(ctx, &rbacmodel.ServiceAccount{
		ID:         "sa-1",
		Meta:       rbacmodel.ServiceAccountV1Meta{Name: "builder", Comments: "build bot", Enabled: true},
		Generation: 1,
	}); err != nil {
		t.Fatalf("seed service account: %v", err)
	}
	if err := rbacrepo.SaveRole(ctx, &rbacmodel.Role{
		ID:         "role-1",
		Meta:       rbacmodel.RoleV1Meta{Name: "admin", Comments: "admin role", Rules: []rbacmodel.PolicyRule{{Resource: "rbac", Verbs: []string{"*"}}}},
		Generation: 1,
	}); err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := rbacrepo.SaveRoleBinding(ctx, &rbacmodel.RoleBinding{
		ID:         "binding-1",
		Meta:       rbacmodel.RoleBindingV1Meta{Name: "binding-1", RoleIDs: []string{"role-1"}, ServiceAccountID: "sa-1", Enabled: true},
		Generation: 1,
	}); err != nil {
		t.Fatalf("seed role binding: %v", err)
	}

	rbacservice.RegisterDiscovery(registry)

	ctx = commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{AllowedAll: true})

	for _, code := range []string{"rbac/serviceaccounts", "rbac/roles", "rbac/rolebindings"} {
		res, err := registry.Lookup(ctx, discoverymodel.LookupRequest{Code: code, Limit: 20})
		if err != nil {
			t.Fatalf("lookup %s: %v", code, err)
		}
		if len(res.Items) != 1 {
			t.Fatalf("unexpected lookup size for %s: %#v", code, res.Items)
		}
	}

	suggestions, err := registry.SuggestResources(ctx, "rbac/")
	if err != nil {
		t.Fatalf("suggest resources: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("expected rbac resource suggestions")
	}
}
