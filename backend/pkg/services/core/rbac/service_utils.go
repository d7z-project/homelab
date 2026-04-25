package rbac

import (
	"context"
	"homelab/pkg/common"
	authservice "homelab/pkg/services/core/auth"

	rbacmodel "homelab/pkg/models/core/rbac"
)

func lockRBAC(ctx context.Context, id string) (func(), error) {
	return common.LockWithTimeout(ctx, "rbac:sync:"+id, 0)
}

func SimulatePermissions(ctx context.Context, saID, verb, resource string) (*rbacmodel.ResourcePermissions, error) {
	return authservice.GetPermissions(ctx, saID, verb, resource)
}
