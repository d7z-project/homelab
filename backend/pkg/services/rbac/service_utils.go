package rbac

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/models"
	authservice "homelab/pkg/services/auth"
)

func lockRBAC(ctx context.Context, id string) (func(), error) {
	return common.LockWithTimeout(ctx, "rbac:sync:"+id, 0)
}

func SimulatePermissions(ctx context.Context, saID, verb, resource string) (*models.ResourcePermissions, error) {
	return authservice.GetPermissions(ctx, saID, verb, resource)
}
