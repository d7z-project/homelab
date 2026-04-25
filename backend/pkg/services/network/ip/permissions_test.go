package ip

import (
	"context"
	"errors"
	"testing"

	commonauth "homelab/pkg/common/auth"
	rbacmodel "homelab/pkg/models/core/rbac"
)

func TestRequireIPResourceAllowsScopedResource(t *testing.T) {
	t.Parallel()

	ctx := commonauth.WithPermissions(context.Background(), &rbacmodel.ResourcePermissions{
		AllowedInstances: []string{ipGroupResource("pool-a")},
	})

	if err := requireIPResource(ctx, ipGroupResource("pool-a")); err != nil {
		t.Fatalf("expected scoped ip permission, got %v", err)
	}
}

func TestRequireIPResourceOrGlobalAllowsGlobalPermission(t *testing.T) {
	t.Parallel()

	ctx := commonauth.WithPermissions(context.Background(), &rbacmodel.ResourcePermissions{
		AllowedInstances: []string{ipResourceBase},
	})

	if err := requireIPResourceOrGlobal(ctx, ipExportResource("exp-a")); err != nil {
		t.Fatalf("expected global ip permission, got %v", err)
	}
}

func TestRequireIPResourceReturnsDeniedResource(t *testing.T) {
	t.Parallel()

	err := requireIPResource(context.Background(), ipExportResource("exp-a"))
	if !errors.Is(err, commonauth.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if err.Error() != "permission denied: network/ip/export/exp-a" {
		t.Fatalf("unexpected error: %v", err)
	}
}
