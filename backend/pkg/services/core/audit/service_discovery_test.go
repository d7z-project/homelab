package audit_test

import (
	"context"
	"testing"
	"time"

	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	auditrepo "homelab/pkg/repositories/core/audit"
	registryruntime "homelab/pkg/runtime/registry"
	auditservice "homelab/pkg/services/core/audit"

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
	common.DB = db

	if err := auditrepo.SaveLog(context.Background(), &models.AuditLog{
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

	auditservice.RegisterDiscovery()

	ctx := commonauth.WithPermissions(context.Background(), &models.ResourcePermissions{AllowedAll: true})

	suggestions, err := registryruntime.Default().SuggestResources(ctx, "audit/")
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
