package auth_test

import (
	"testing"

	rbacmodel "homelab/pkg/models/core/rbac"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	authservice "homelab/pkg/services/core/auth"
	"homelab/pkg/testkit"
)

func TestGetPermissionsMatchesResourceRules(t *testing.T) {
	t.Parallel()

	deps := testkit.NewModuleDeps(t)
	rbacrepo.Configure(deps.DB)
	rbacrepo.ClearCache()
	ctx := t.Context()

	if err := rbacrepo.SaveServiceAccount(ctx, &rbacmodel.ServiceAccount{
		ID: "sa-1",
		Meta: rbacmodel.ServiceAccountV1Meta{
			Name:    "sa-1",
			Enabled: true,
		},
		Status: rbacmodel.ServiceAccountV1Status{
			HasAuthSecret: true,
		},
		Generation: 1,
	}); err != nil {
		t.Fatalf("seed service account: %v", err)
	}

	if err := rbacrepo.SaveRole(ctx, &rbacmodel.Role{
		ID: "role-1",
		Meta: rbacmodel.RoleV1Meta{
			Name: "role-1",
			Rules: []rbacmodel.PolicyRule{
				{Resource: "*", Verbs: []string{"admin"}},
				{Resource: "network/dns/domain/*", Verbs: []string{"list"}},
				{Resource: "network/dns/domain/example.com", Verbs: []string{"create"}},
				{Resource: "network/dns/domain/example.com/record/name/www/type/A", Verbs: []string{"update"}},
			},
		},
		Generation: 1,
	}); err != nil {
		t.Fatalf("seed role: %v", err)
	}

	if err := rbacrepo.SaveRoleBinding(ctx, &rbacmodel.RoleBinding{
		ID: "binding-1",
		Meta: rbacmodel.RoleBindingV1Meta{
			Name:             "binding-1",
			ServiceAccountID: "sa-1",
			RoleIDs:          []string{"role-1"},
			Enabled:          true,
		},
		Generation: 1,
	}); err != nil {
		t.Fatalf("seed role binding: %v", err)
	}

	tests := []struct {
		name            string
		verb            string
		resource        string
		wantAllowedAll  bool
		wantInstances   []string
		wantMatchedRule string
	}{
		{
			name:            "global wildcard grants full access",
			verb:            "admin",
			resource:        "rbac",
			wantAllowedAll:  true,
			wantMatchedRule: "*",
		},
		{
			name:            "prefix wildcard contributes domain-level instances at route scope",
			verb:            "list",
			resource:        "network/dns",
			wantInstances:   []string{"network/dns/domain"},
			wantMatchedRule: "network/dns/domain/*",
		},
		{
			name:            "instance rule contributes allowed instances at route scope",
			verb:            "create",
			resource:        "network/dns",
			wantInstances:   []string{"network/dns/domain/example.com"},
			wantMatchedRule: "network/dns/domain/example.com",
		},
		{
			name:            "nested instance rule contributes nested allowed instance",
			verb:            "update",
			resource:        "network/dns",
			wantInstances:   []string{"network/dns/domain/example.com/record/name/www/type/A"},
			wantMatchedRule: "network/dns/domain/example.com/record/name/www/type/A",
		},
		{
			name:            "exact resource grants full access inside service checks",
			verb:            "create",
			resource:        "network/dns/domain/example.com",
			wantAllowedAll:  true,
			wantMatchedRule: "network/dns/domain/example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms, err := authservice.GetPermissions(ctx, "sa-1", tt.verb, tt.resource)
			if err != nil {
				t.Fatalf("GetPermissions(%s, %s): %v", tt.verb, tt.resource, err)
			}
			if perms.AllowedAll != tt.wantAllowedAll {
				t.Fatalf("AllowedAll mismatch: got %v want %v", perms.AllowedAll, tt.wantAllowedAll)
			}
			if len(perms.AllowedInstances) != len(tt.wantInstances) {
				t.Fatalf("AllowedInstances size mismatch: got %#v want %#v", perms.AllowedInstances, tt.wantInstances)
			}
			for i := range tt.wantInstances {
				if perms.AllowedInstances[i] != tt.wantInstances[i] {
					t.Fatalf("AllowedInstances mismatch: got %#v want %#v", perms.AllowedInstances, tt.wantInstances)
				}
			}
			if perms.MatchedRule == nil || perms.MatchedRule.Resource != tt.wantMatchedRule {
				t.Fatalf("MatchedRule mismatch: got %#v want %q", perms.MatchedRule, tt.wantMatchedRule)
			}
		})
	}
}
