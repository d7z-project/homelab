package site_test

import (
	"net/http"
	"testing"

	rbacmodel "homelab/pkg/models/core/rbac"
	moduleauth "homelab/pkg/modules/core/auth"
	modulesecret "homelab/pkg/modules/core/secret"
	modulesite "homelab/pkg/modules/network/site"
	siterepo "homelab/pkg/repositories/network/site"
	"homelab/pkg/testkit"
)

func TestSiteModuleCRUDAndPermissions(t *testing.T) {
	env := testkit.StartApp(t, modulesecret.New(), moduleauth.New(), modulesite.New(nil))
	rootToken := testkit.RootToken(t, env)

	createGroup := env.DoJSON(http.MethodPost, "/api/v1/network/site/pools", rootToken, map[string]any{
		"id": "site-1",
		"meta": map[string]any{
			"name": "keywords",
		},
	})
	testkit.MustStatus(t, createGroup, http.StatusOK)

	scanGroups := env.DoJSON(http.MethodGet, "/api/v1/network/site/pools", rootToken, nil)
	testkit.MustStatus(t, scanGroups, http.StatusOK)
	scanBody := testkit.DecodeJSON[struct {
		Items []map[string]any `json:"items"`
	}](t, scanGroups)
	if len(scanBody.Items) != 1 {
		t.Fatalf("expected 1 site group, got %d", len(scanBody.Items))
	}

	group, err := siterepo.GetGroup(env.Context(), "site-1")
	if err != nil || group == nil {
		t.Fatalf("expected persisted site group, got %v %v", group, err)
	}

	deniedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-site-denied", "site denied")
	if err != nil {
		t.Fatalf("seed denied service account: %v", err)
	}
	deniedCreate := env.DoJSON(http.MethodPost, "/api/v1/network/site/pools", deniedToken, map[string]any{
		"id": "site-2",
		"meta": map[string]any{
			"name": "blocked",
		},
	})
	testkit.MustStatus(t, deniedCreate, http.StatusUnauthorized)

	allowedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-site-read", "site read",
		rbacmodel.PolicyRule{Resource: "network/site", Verbs: []string{"list"}},
	)
	if err != nil {
		t.Fatalf("seed allowed service account: %v", err)
	}
	allowedList := env.DoJSON(http.MethodGet, "/api/v1/network/site/pools", allowedToken, nil)
	testkit.MustStatus(t, allowedList, http.StatusOK)
}
