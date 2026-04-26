package rbac_test

import (
	"net/http"
	"testing"

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
