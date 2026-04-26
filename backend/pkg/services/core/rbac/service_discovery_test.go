package rbac_test

import (
	"testing"

	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	rbacservice "homelab/pkg/services/core/rbac"
	"homelab/pkg/testkit"
)

func TestRegisterDiscovery(t *testing.T) {
	t.Parallel()

	deps := testkit.NewModuleDeps(t)
	rbacrepo.Configure(deps.DB)
	ctx := t.Context()

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

	rbacservice.RegisterDiscovery(deps.Registry)

	ctx = commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{AllowedAll: true})

	for _, code := range []string{"rbac/serviceaccounts", "rbac/roles", "rbac/rolebindings"} {
		res, err := deps.Registry.Lookup(ctx, discoverymodel.LookupRequest{Code: code, Limit: 20})
		if err != nil {
			t.Fatalf("lookup %s: %v", code, err)
		}
		if len(res.Items) != 1 {
			t.Fatalf("unexpected lookup size for %s: %#v", code, res.Items)
		}
	}

	suggestions, err := deps.Registry.SuggestResources(ctx, "rbac/")
	if err != nil {
		t.Fatalf("suggest resources: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("expected rbac resource suggestions")
	}
}
