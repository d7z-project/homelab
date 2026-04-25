package controllers

import (
	"context"
	"errors"
	"testing"

	commonauth "homelab/pkg/common/auth"
	rbacmodel "homelab/pkg/models/core/rbac"
)

func TestRequireScopedPermissionAllowsInstanceScope(t *testing.T) {
	t.Parallel()

	ctx := commonauth.WithPermissions(context.Background(), &rbacmodel.ResourcePermissions{
		MatchedRule:      &rbacmodel.PolicyRule{},
		AllowedInstances: []string{"network/ip/pool-a"},
	})

	if err := RequireScopedPermission(ctx, NetworkIPResourceBase, "pool-a"); err != nil {
		t.Fatalf("expected instance permission to pass, got %v", err)
	}
}

func TestRequireScopedPermissionAllowsGlobalScope(t *testing.T) {
	t.Parallel()

	ctx := commonauth.WithPermissions(context.Background(), &rbacmodel.ResourcePermissions{
		AllowedAll: true,
	})

	if err := RequireScopedPermission(ctx, NetworkSiteResourceBase, "site-a"); err != nil {
		t.Fatalf("expected global permission to pass, got %v", err)
	}
}

func TestRequireScopedPermissionReturnsDeniedResource(t *testing.T) {
	t.Parallel()

	err := RequireScopedPermission(context.Background(), NetworkIPResourceBase, "pool-a")
	if !errors.Is(err, commonauth.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if err.Error() != "permission denied: network/ip/pool-a" {
		t.Fatalf("unexpected error: %v", err)
	}
}
