package site

import (
	"context"
	"fmt"

	commonauth "homelab/pkg/common/auth"
)

const siteResourceBase = "network/site"

func siteGroupResource(id string) string {
	return siteResourceBase + "/" + id
}

func requireSiteResource(ctx context.Context, resource string) error {
	if commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil
	}
	return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
}

func requireSiteResourceOrGlobal(ctx context.Context, resource string) error {
	perms := commonauth.PermissionsFromContext(ctx)
	if perms.IsAllowed(resource) || perms.IsAllowed(siteResourceBase) {
		return nil
	}
	return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
}
