package rbac

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	rbacrepo "homelab/pkg/repositories/rbac"
	"homelab/pkg/services/discovery"
	"strings"
)

func init() {
	discovery.Register("rbac/serviceaccounts", func(ctx context.Context, search string, cursor string, limit int) (*models.PaginationResponse[models.LookupItem], error) {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
			return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
		}
		res, err := rbacrepo.ScanServiceAccounts(ctx, cursor, limit, search)
		if err != nil {
			return nil, err
		}
		var items []models.LookupItem
		for _, sa := range res.Items {
			items = append(items, models.LookupItem{
				ID:          sa.ID,
				Name:        sa.Name,
				Description: sa.Comments,
			})
		}
		return &models.PaginationResponse[models.LookupItem]{
			Items:      items,
			NextCursor: res.NextCursor,
			HasMore:    res.HasMore,
			Total:      res.Total,
		}, nil
	})

	discovery.Register("rbac/roles", func(ctx context.Context, search string, cursor string, limit int) (*models.PaginationResponse[models.LookupItem], error) {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
			return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
		}
		res, err := rbacrepo.ScanRoles(ctx, cursor, limit, search)
		if err != nil {
			return nil, err
		}
		var items []models.LookupItem
		for _, r := range res.Items {
			items = append(items, models.LookupItem{
				ID:          r.ID,
				Name:        r.Name,
				Description: r.Comments,
			})
		}
		return &models.PaginationResponse[models.LookupItem]{
			Items:      items,
			NextCursor: res.NextCursor,
			HasMore:    res.HasMore,
			Total:      res.Total,
		}, nil
	})

	discovery.Register("rbac/rolebindings", func(ctx context.Context, search string, cursor string, limit int) (*models.PaginationResponse[models.LookupItem], error) {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
			return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
		}
		res, err := rbacrepo.ScanRoleBindings(ctx, cursor, limit, search)
		if err != nil {
			return nil, err
		}
		var items []models.LookupItem
		for _, rb := range res.Items {
			items = append(items, models.LookupItem{
				ID:          rb.ID,
				Name:        rb.ID, // Binding ID is usually its unique name
				Description: fmt.Sprintf("%s -> %s", rb.ServiceAccountID, strings.Join(rb.RoleIDs, ", ")),
			})
		}
		return &models.PaginationResponse[models.LookupItem]{
			Items:      items,
			NextCursor: res.NextCursor,
			HasMore:    res.HasMore,
			Total:      res.Total,
		}, nil
	})
}

func ScanServiceAccounts(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.ServiceAccount], error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}
	return rbacrepo.ScanServiceAccounts(ctx, cursor, limit, search)
}

func ScanRoles(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.Role], error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}
	return rbacrepo.ScanRoles(ctx, cursor, limit, search)
}

func ScanRoleBindings(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.RoleBinding], error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}
	return rbacrepo.ScanRoleBindings(ctx, cursor, limit, search)
}
