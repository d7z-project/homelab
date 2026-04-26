package audit_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	auditmodel "homelab/pkg/models/core/audit"
	rbacmodel "homelab/pkg/models/core/rbac"
	moduleaudit "homelab/pkg/modules/core/audit"
	moduleauth "homelab/pkg/modules/core/auth"
	auditrepo "homelab/pkg/repositories/core/audit"
	runtimepkg "homelab/pkg/runtime"
	"homelab/pkg/testkit"
)

func TestAuditModuleScanLogsAndPermissions(t *testing.T) {
	env := testkit.StartApp(t,
		testkit.SeedModule("test.seed.audit.logs", func(ctx context.Context, _ runtimepkg.ModuleDeps) error {
			return auditrepo.SaveLog(ctx, &auditmodel.AuditLog{
				ID:        "log-1",
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Subject:   "root",
				Action:    "CREATE",
				Resource:  "audit",
				TargetID:  "builder",
				Message:   "created builder",
				Status:    "Success",
			})
		}),
		moduleauth.New(),
		moduleaudit.New(),
	)

	rootToken := testkit.RootToken(t, env)

	rootResp := env.DoJSON(http.MethodGet, "/api/v1/audit/logs", rootToken, nil)
	testkit.MustStatus(t, rootResp, http.StatusOK)
	rootBody := testkit.DecodeJSON[struct {
		Items []map[string]any `json:"items"`
	}](t, rootResp)
	found := false
	for _, item := range rootBody.Items {
		if item["id"] == "log-1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected seeded audit log in response, got %v", rootBody.Items)
	}

	deniedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-audit-denied", "audit denied")
	if err != nil {
		t.Fatalf("seed denied service account: %v", err)
	}
	deniedResp := env.DoJSON(http.MethodGet, "/api/v1/audit/logs", deniedToken, nil)
	testkit.MustStatus(t, deniedResp, http.StatusForbidden)

	allowedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-audit-read", "audit read",
		rbacmodel.PolicyRule{Resource: "audit", Verbs: []string{"list"}},
	)
	if err != nil {
		t.Fatalf("seed allowed service account: %v", err)
	}
	allowedResp := env.DoJSON(http.MethodGet, "/api/v1/audit/logs", allowedToken, nil)
	testkit.MustStatus(t, allowedResp, http.StatusOK)
}

func TestAuditModuleCleanupVerbMapping(t *testing.T) {
	env := testkit.StartApp(t, moduleauth.New(), moduleaudit.New())

	listToken, err := testkit.SeedServiceAccount(env.Context(), "sa-audit-list-only", "audit list only",
		rbacmodel.PolicyRule{Resource: "audit", Verbs: []string{"list"}},
	)
	if err != nil {
		t.Fatalf("seed list-only service account: %v", err)
	}

	listResp := env.DoJSON(http.MethodPost, "/api/v1/audit/logs/cleanup?days=1", listToken, nil)
	testkit.MustStatus(t, listResp, http.StatusForbidden)
	if got := listResp.Header().Get("X-Matched-Policy"); got != "" {
		t.Fatalf("expected route-level denial for cleanup without delete, got %q", got)
	}

	deleteToken, err := testkit.SeedServiceAccount(env.Context(), "sa-audit-delete", "audit delete",
		rbacmodel.PolicyRule{Resource: "audit", Verbs: []string{"delete"}},
	)
	if err != nil {
		t.Fatalf("seed delete service account: %v", err)
	}

	deleteResp := env.DoJSON(http.MethodPost, "/api/v1/audit/logs/cleanup?days=1", deleteToken, nil)
	testkit.MustStatus(t, deleteResp, http.StatusForbidden)
	if got := deleteResp.Header().Get("X-Matched-Policy"); got != "audit" {
		t.Fatalf("expected route pass then service denial for cleanup, got matched policy %q", got)
	}
}
