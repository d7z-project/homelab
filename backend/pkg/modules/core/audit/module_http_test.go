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
	if len(rootBody.Items) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(rootBody.Items))
	}

	deniedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-audit-denied", "audit denied")
	if err != nil {
		t.Fatalf("seed denied service account: %v", err)
	}
	deniedResp := env.DoJSON(http.MethodGet, "/api/v1/audit/logs", deniedToken, nil)
	testkit.MustStatus(t, deniedResp, http.StatusUnauthorized)

	allowedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-audit-read", "audit read",
		rbacmodel.PolicyRule{Resource: "audit", Verbs: []string{"list"}},
	)
	if err != nil {
		t.Fatalf("seed allowed service account: %v", err)
	}
	allowedResp := env.DoJSON(http.MethodGet, "/api/v1/audit/logs", allowedToken, nil)
	testkit.MustStatus(t, allowedResp, http.StatusOK)
}
