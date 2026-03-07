package rbac

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	rbacrepo "homelab/pkg/repositories/rbac"
	"homelab/pkg/services/discovery"
)

func init() {
	discovery.Register("rbac/serviceaccounts", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
			return nil, 0, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
		}
		if limit <= 0 {
			limit = 20
		}
		sas, total, err := rbacrepo.ListServiceAccounts(ctx, uint64(offset/limit), uint(limit), search)
		if err != nil {
			return nil, 0, err
		}
		var items []models.LookupItem
		for _, sa := range sas {
			items = append(items, models.LookupItem{
				ID:          sa.ID,
				Name:        sa.Name,
				Description: sa.Comments,
			})
		}
		return items, int(total), nil
	})

	discovery.Register("rbac/roles", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
			return nil, 0, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
		}
		if limit <= 0 {
			limit = 20
		}
		roles, total, err := rbacrepo.ListRoles(ctx, uint64(offset/limit), uint(limit), search)
		if err != nil {
			return nil, 0, err
		}
		var items []models.LookupItem
		for _, r := range roles {
			items = append(items, models.LookupItem{
				ID:          r.ID,
				Name:        r.Name,
				Description: r.Comments,
			})
		}
		return items, int(total), nil
	})
}
