package audit_test

import (
	"context"
	"testing"
	"time"

	commonauth "homelab/pkg/common/auth"
	auditmodel "homelab/pkg/models/core/audit"
	rbacmodel "homelab/pkg/models/core/rbac"
	auditrepo "homelab/pkg/repositories/core/audit"
	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"
	auditservice "homelab/pkg/services/core/audit"

	"github.com/spf13/afero"
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
	registry := registryruntime.New()
	deps := runtimepkg.ModuleDeps{
		Dependencies: runtimepkg.Dependencies{
			DB:     db,
			FS:     afero.NewMemMapFs(),
			TempFS: afero.NewMemMapFs(),
		},
		Registry: registry,
	}
	ctx := deps.WithContext(context.Background())

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

	auditservice.RegisterDiscovery(registry)

	ctx = commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{AllowedAll: true})

	suggestions, err := registry.SuggestResources(ctx, "audit/")
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
