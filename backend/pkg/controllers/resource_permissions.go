package controllers

import (
	"context"
	"fmt"

	commonauth "homelab/pkg/common/auth"
)

const (
	NetworkIPResourceBase   = "network/ip"
	NetworkSiteResourceBase = "network/site"
)

func RequireScopedPermission(ctx context.Context, base string, id string) error {
	resource := scopedResource(base, id)
	perms := commonauth.PermissionsFromContext(ctx)
	if perms.IsAllowed(resource) || perms.IsAllowed(base) {
		return nil
	}
	return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
}

func scopedResource(base string, id string) string {
	return base + "/" + id
}
