package site

import (
	"context"
	"errors"
	"testing"

	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
)

func TestRequireSiteResourceAllowsScopedResource(t *testing.T) {
	t.Parallel()

	ctx := commonauth.WithPermissions(context.Background(), &models.ResourcePermissions{
		AllowedInstances: []string{siteGroupResource("site-a")},
	})

	if err := requireSiteResource(ctx, siteGroupResource("site-a")); err != nil {
		t.Fatalf("expected scoped site permission, got %v", err)
	}
}

func TestRequireSiteResourceOrGlobalAllowsGlobalPermission(t *testing.T) {
	t.Parallel()

	ctx := commonauth.WithPermissions(context.Background(), &models.ResourcePermissions{
		AllowedInstances: []string{siteResourceBase},
	})

	if err := requireSiteResourceOrGlobal(ctx, siteGroupResource("site-a")); err != nil {
		t.Fatalf("expected global site permission, got %v", err)
	}
}

func TestRequireSiteResourceReturnsDeniedResource(t *testing.T) {
	t.Parallel()

	err := requireSiteResource(context.Background(), siteGroupResource("site-a"))
	if !errors.Is(err, commonauth.ErrPermissionDenied) {
		t.Fatalf("expected permission denied, got %v", err)
	}
	if err.Error() != "permission denied: network/site/site-a" {
		t.Fatalf("unexpected error: %v", err)
	}
}
