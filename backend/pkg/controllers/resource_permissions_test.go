package controllers

import (
	"context"
	"errors"
	"testing"

	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
)

func TestRequireScopedPermissionAllowsInstanceScope(t *testing.T) {
	t.Parallel()

	ctx := commonauth.WithPermissions(context.Background(), &models.ResourcePermissions{
		MatchedRule:      &models.PolicyRule{},
		AllowedInstances: []string{"network/ip/pool-a"},
	})

	if err := requireScopedPermission(ctx, networkIPResourceBase, "pool-a"); err != nil {
		t.Fatalf("expected instance permission to pass, got %v", err)
	}
}

func TestRequireScopedPermissionAllowsGlobalScope(t *testing.T) {
	t.Parallel()

	ctx := commonauth.WithPermissions(context.Background(), &models.ResourcePermissions{
		AllowedAll: true,
	})

	if err := requireScopedPermission(ctx, networkSiteResourceBase, "site-a"); err != nil {
		t.Fatalf("expected global permission to pass, got %v", err)
	}
}

func TestRequireScopedPermissionReturnsDeniedResource(t *testing.T) {
	t.Parallel()

	err := requireScopedPermission(context.Background(), networkIPResourceBase, "pool-a")
	if !errors.Is(err, commonauth.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if err.Error() != "permission denied: network/ip/pool-a" {
		t.Fatalf("unexpected error: %v", err)
	}
}
