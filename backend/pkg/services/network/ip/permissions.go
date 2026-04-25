package ip

import (
	"context"
	"fmt"

	commonauth "homelab/pkg/common/auth"
)

const (
	ipResourceBase       = "network/ip"
	ipExportResourceBase = "network/ip/export"
)

func ipGroupResource(id string) string {
	return ipResourceBase + "/" + id
}

func ipExportResource(id string) string {
	return ipExportResourceBase + "/" + id
}

func requireIPResource(ctx context.Context, resource string) error {
	if commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil
	}
	return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
}

func requireIPResourceOrGlobal(ctx context.Context, resource string) error {
	perms := commonauth.PermissionsFromContext(ctx)
	if perms.IsAllowed(resource) || perms.IsAllowed(ipResourceBase) {
		return nil
	}
	return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
}
