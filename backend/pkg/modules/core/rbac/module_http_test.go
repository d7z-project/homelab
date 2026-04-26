package rbac_test

import (
	"net/http"
	"testing"

	rbacmodel "homelab/pkg/models/core/rbac"
	moduleauth "homelab/pkg/modules/core/auth"
	modulerbac "homelab/pkg/modules/core/rbac"
	modulesecret "homelab/pkg/modules/core/secret"
	"homelab/pkg/testkit"
)

func TestRBACModuleCRUDAndPermissions(t *testing.T) {
	env := testkit.StartApp(t, modulesecret.New(), moduleauth.New(), modulerbac.New())
	rootToken := testkit.RootToken(t, env)

	createRole := env.DoJSON(http.MethodPost, "/api/v1/rbac/roles", rootToken, map[string]any{
		"meta": map[string]any{
			"name": "rbac reader",
			"rules": []map[string]any{
				{"resource": "rbac", "verbs": []string{"list"}},
			},
		},
	})
	testkit.MustStatus(t, createRole, http.StatusOK)
	roleBody := testkit.DecodeJSON[struct {
		ID string `json:"id"`
	}](t, createRole)
	if roleBody.ID == "" {
		t.Fatalf("expected created role id")
	}

	createSA := env.DoJSON(http.MethodPost, "/api/v1/rbac/serviceaccounts", rootToken, map[string]any{
		"id": "sa-rbac-reader",
		"meta": map[string]any{
			"name":    "rbac reader",
			"enabled": true,
		},
	})
	testkit.MustStatus(t, createSA, http.StatusOK)
	saBody := testkit.DecodeJSON[struct {
		Token          string `json:"token"`
		ServiceAccount struct {
			ID     string `json:"id"`
			Status struct {
				HasAuthSecret bool `json:"hasAuthSecret"`
			} `json:"status"`
		} `json:"serviceAccount"`
	}](t, createSA)
	if saBody.Token == "" || !saBody.ServiceAccount.Status.HasAuthSecret {
		t.Fatalf("expected service account token response with auth secret")
	}

	createBinding := env.DoJSON(http.MethodPost, "/api/v1/rbac/rolebindings", rootToken, map[string]any{
		"meta": map[string]any{
			"name":             "rbac binding",
			"serviceAccountId": "sa-rbac-reader",
			"roleIds":          []string{roleBody.ID},
			"enabled":          true,
		},
	})
	testkit.MustStatus(t, createBinding, http.StatusOK)

	saList := env.DoJSON(http.MethodGet, "/api/v1/rbac/serviceaccounts", saBody.Token, nil)
	testkit.MustStatus(t, saList, http.StatusOK)

	deniedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-rbac-denied", "rbac denied")
	if err != nil {
		t.Fatalf("seed denied service account: %v", err)
	}
	deniedList := env.DoJSON(http.MethodGet, "/api/v1/rbac/serviceaccounts", deniedToken, nil)
	testkit.MustStatus(t, deniedList, http.StatusForbidden)
}

func TestRBACModuleVerbMappings(t *testing.T) {
	env := testkit.StartApp(t, modulesecret.New(), moduleauth.New(), modulerbac.New())
	rootToken := testkit.RootToken(t, env)

	createTarget := env.DoJSON(http.MethodPost, "/api/v1/rbac/serviceaccounts", rootToken, map[string]any{
		"id": "sa-reset-target",
		"meta": map[string]any{
			"name":    "reset target",
			"enabled": true,
		},
	})
	testkit.MustStatus(t, createTarget, http.StatusOK)

	simulateToken, err := testkit.SeedServiceAccount(env.Context(), "sa-rbac-simulate", "rbac simulate",
		rbacmodel.PolicyRule{Resource: "rbac", Verbs: []string{"simulate"}},
	)
	if err != nil {
		t.Fatalf("seed simulate service account: %v", err)
	}

	simulateResp := env.DoJSON(http.MethodPost, "/api/v1/rbac/simulate", simulateToken, map[string]any{
		"serviceAccountId": "sa-rbac-simulate",
		"verb":             "list",
		"resource":         "rbac",
	})
	testkit.MustStatus(t, simulateResp, http.StatusOK)
	if got := simulateResp.Header().Get("X-Matched-Policy"); got != "rbac" {
		t.Fatalf("unexpected matched policy for simulate: %q", got)
	}

	simulateDeniedList := env.DoJSON(http.MethodGet, "/api/v1/rbac/serviceaccounts", simulateToken, nil)
	testkit.MustStatus(t, simulateDeniedList, http.StatusForbidden)
	if got := simulateDeniedList.Header().Get("X-Matched-Policy"); got != "" {
		t.Fatalf("expected route-level denial for list without permission, got %q", got)
	}

	updateToken, err := testkit.SeedServiceAccount(env.Context(), "sa-rbac-update", "rbac update",
		rbacmodel.PolicyRule{Resource: "rbac", Verbs: []string{"update"}},
	)
	if err != nil {
		t.Fatalf("seed update service account: %v", err)
	}

	resetResp := env.DoJSON(http.MethodPost, "/api/v1/rbac/serviceaccounts/sa-reset-target/reset", updateToken, nil)
	testkit.MustStatus(t, resetResp, http.StatusOK)
	if got := resetResp.Header().Get("X-Matched-Policy"); got != "rbac" {
		t.Fatalf("unexpected matched policy for reset: %q", got)
	}

	updateDeniedSimulate := env.DoJSON(http.MethodPost, "/api/v1/rbac/simulate", updateToken, map[string]any{
		"serviceAccountId": "sa-rbac-update",
		"verb":             "list",
		"resource":         "rbac",
	})
	testkit.MustStatus(t, updateDeniedSimulate, http.StatusForbidden)
	if got := updateDeniedSimulate.Header().Get("X-Matched-Policy"); got != "" {
		t.Fatalf("expected route-level denial for simulate without permission, got %q", got)
	}

	listToken, err := testkit.SeedServiceAccount(env.Context(), "sa-rbac-list", "rbac list",
		rbacmodel.PolicyRule{Resource: "rbac", Verbs: []string{"list"}},
	)
	if err != nil {
		t.Fatalf("seed list service account: %v", err)
	}

	suggestResp := env.DoJSON(http.MethodGet, "/api/v1/rbac/resources/suggest?prefix=rbac", listToken, nil)
	testkit.MustStatus(t, suggestResp, http.StatusOK)
	if got := suggestResp.Header().Get("X-Matched-Policy"); got != "rbac" {
		t.Fatalf("unexpected matched policy for resource suggestion: %q", got)
	}
}
