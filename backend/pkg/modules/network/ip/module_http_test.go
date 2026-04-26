package ip_test

import (
	"net/http"
	"testing"

	rbacmodel "homelab/pkg/models/core/rbac"
	moduleauth "homelab/pkg/modules/core/auth"
	modulesecret "homelab/pkg/modules/core/secret"
	moduleip "homelab/pkg/modules/network/ip"
	iprepo "homelab/pkg/repositories/network/ip"
	"homelab/pkg/testkit"
)

func TestIPModuleCRUDAndPermissions(t *testing.T) {
	env := testkit.StartApp(t, modulesecret.New(), moduleauth.New(), moduleip.New(nil))
	rootToken := testkit.RootToken(t, env)

	createPool := env.DoJSON(http.MethodPost, "/api/v1/network/ip/pools", rootToken, map[string]any{
		"id": "pool-1",
		"meta": map[string]any{
			"name": "internal",
		},
	})
	testkit.MustStatus(t, createPool, http.StatusOK)

	scanPools := env.DoJSON(http.MethodGet, "/api/v1/network/ip/pools", rootToken, nil)
	testkit.MustStatus(t, scanPools, http.StatusOK)
	scanBody := testkit.DecodeJSON[struct {
		Items []map[string]any `json:"items"`
	}](t, scanPools)
	if len(scanBody.Items) != 1 {
		t.Fatalf("expected 1 ip pool, got %d", len(scanBody.Items))
	}

	pool, err := iprepo.GetPool(env.Context(), "pool-1")
	if err != nil || pool == nil {
		t.Fatalf("expected persisted ip pool, got %v %v", pool, err)
	}

	deniedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-ip-denied", "ip denied")
	if err != nil {
		t.Fatalf("seed denied service account: %v", err)
	}
	deniedCreate := env.DoJSON(http.MethodPost, "/api/v1/network/ip/pools", deniedToken, map[string]any{
		"id": "pool-2",
		"meta": map[string]any{
			"name": "blocked",
		},
	})
	testkit.MustStatus(t, deniedCreate, http.StatusUnauthorized)

	allowedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-ip-read", "ip read",
		rbacmodel.PolicyRule{Resource: "network/ip", Verbs: []string{"list"}},
	)
	if err != nil {
		t.Fatalf("seed allowed service account: %v", err)
	}
	allowedList := env.DoJSON(http.MethodGet, "/api/v1/network/ip/pools", allowedToken, nil)
	testkit.MustStatus(t, allowedList, http.StatusOK)
}
