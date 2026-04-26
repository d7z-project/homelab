package audit_test

import (
	"testing"
	"time"

	commonauth "homelab/pkg/common/auth"
	auditmodel "homelab/pkg/models/core/audit"
	rbacmodel "homelab/pkg/models/core/rbac"
	auditrepo "homelab/pkg/repositories/core/audit"
	auditservice "homelab/pkg/services/core/audit"
	"homelab/pkg/testkit"
)

func TestRegisterDiscovery(t *testing.T) {
	t.Parallel()

	deps := testkit.NewModuleDeps(t)
	auditrepo.Configure(deps.DB)
	ctx := t.Context()

	if err := auditrepo.SaveLog(ctx, &auditmodel.AuditLog{
		ID:        "log-1",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Subject:   "root",
		Action:    "CREATE",
		Resource:  "RBAC/ServiceAccount",
		TargetID:  "builder",
		Message:   "created builder",
		Status:    "Success",
	}); err != nil {
		t.Fatalf("seed audit log: %v", err)
	}

	auditservice.RegisterDiscovery(deps.Registry)

	ctx = commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{AllowedAll: true})

	suggestions, err := deps.Registry.SuggestResources(ctx, "audit/")
	if err != nil {
		t.Fatalf("suggest resources: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("expected audit resource suggestions")
	}

	res, err := auditservice.ScanLogs(ctx, "", 20, "")
	if err != nil {
		t.Fatalf("scan logs: %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("expected 1 audit log, got %d", len(res.Items))
	}
}
