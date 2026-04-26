package rbac

import (
	"context"
	"fmt"
	"strings"

	commonauth "homelab/pkg/common/auth"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	registryruntime "homelab/pkg/runtime/registry"

	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	"homelab/pkg/models/shared"
)

func RegisterDiscovery(registry *registryruntime.Registry) {
	if registry == nil {
		return
	}
	registerResourceDiscovery(registry)

	_ = registry.RegisterLookup("rbac/serviceaccounts", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
			return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
		}
		res, err := rbacrepo.ScanServiceAccounts(ctx, cursor, limit, search)
		if err != nil {
			return nil, err
		}
		var items []discoverymodel.LookupItem
		for _, sa := range res.Items {
			items = append(items, discoverymodel.LookupItem{
				ID:          sa.ID,
				Name:        sa.Meta.Name,
				Description: sa.Meta.Comments,
			})
		}
		return &shared.PaginationResponse[discoverymodel.LookupItem]{
			Items:      items,
			NextCursor: res.NextCursor,
			HasMore:    res.HasMore,
		}, nil
	})

	_ = registry.RegisterLookup("rbac/roles", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
			return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
		}
		res, err := rbacrepo.ScanRoles(ctx, cursor, limit, search)
		if err != nil {
			return nil, err
		}
		var items []discoverymodel.LookupItem
		for _, r := range res.Items {
			items = append(items, discoverymodel.LookupItem{
				ID:          r.ID,
				Name:        r.Meta.Name,
				Description: r.Meta.Comments,
			})
		}
		return &shared.PaginationResponse[discoverymodel.LookupItem]{
			Items:      items,
			NextCursor: res.NextCursor,
			HasMore:    res.HasMore,
		}, nil
	})

	_ = registry.RegisterLookup("rbac/rolebindings", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
			return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
		}
		res, err := rbacrepo.ScanRoleBindings(ctx, cursor, limit, search)
		if err != nil {
			return nil, err
		}
		var items []discoverymodel.LookupItem
		for _, rb := range res.Items {
			items = append(items, discoverymodel.LookupItem{
				ID:          rb.ID,
				Name:        rb.ID,
				Description: fmt.Sprintf("%s -> %s", rb.Meta.ServiceAccountID, strings.Join(rb.Meta.RoleIDs, ", ")),
			})
		}
		return &shared.PaginationResponse[discoverymodel.LookupItem]{
			Items:      items,
			NextCursor: res.NextCursor,
			HasMore:    res.HasMore,
		}, nil
	})
}

func ScanServiceAccounts(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[rbacmodel.ServiceAccount], error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}
	return rbacrepo.ScanServiceAccounts(ctx, cursor, limit, search)
}

func ScanRoles(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[rbacmodel.Role], error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}
	return rbacrepo.ScanRoles(ctx, cursor, limit, search)
}

func ScanRoleBindings(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[rbacmodel.RoleBinding], error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}
	return rbacrepo.ScanRoleBindings(ctx, cursor, limit, search)
}
