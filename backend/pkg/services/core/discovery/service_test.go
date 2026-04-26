package discovery_test

import (
	"testing"

	commonauth "homelab/pkg/common/auth"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	discoveryservice "homelab/pkg/services/core/discovery"
	rbacservice "homelab/pkg/services/core/rbac"
	"homelab/pkg/testkit"

	rbacmodel "homelab/pkg/models/core/rbac"
)

func TestScanCodesFiltersInaccessibleLookups(t *testing.T) {
	t.Parallel()

	deps := testkit.NewModuleDeps(t)
	rbacrepo.Configure(deps.DB)
	ctx := t.Context()

	rbacservice.RegisterDiscovery(deps.Registry)
	service := discoveryservice.NewService(deps)

	rootCtx := commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{AllowedAll: true})
	rootCodes, err := service.ScanCodes(rootCtx)
	if err != nil {
		t.Fatalf("scan codes as root: %v", err)
	}
	if len(rootCodes) == 0 {
		t.Fatal("expected visible discovery codes for root context")
	}

	limitedCtx := commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{})
	limitedCodes, err := service.ScanCodes(limitedCtx)
	if err != nil {
		t.Fatalf("scan codes as limited identity: %v", err)
	}
	if len(limitedCodes) != 0 {
		t.Fatalf("expected inaccessible codes to be filtered, got %#v", limitedCodes)
	}
}
