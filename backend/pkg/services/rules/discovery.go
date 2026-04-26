package rules

import (
	"context"
	"strings"

	metav1 "homelab/pkg/apis/meta/v1"
	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	"homelab/pkg/models/shared"
	iprepo "homelab/pkg/repositories/network/ip"
	siterepo "homelab/pkg/repositories/network/site"
	registryruntime "homelab/pkg/runtime/registry"
)

func RegisterDiscovery(registry *registryruntime.Registry) {
	RegisterIPDiscovery(registry)
	RegisterSiteDiscovery(registry)
}

func RegisterIPDiscovery(registry *registryruntime.Registry) {
	registerIPDiscovery(registry)
}

func RegisterSiteDiscovery(registry *registryruntime.Registry) {
	registerSiteDiscovery(registry)
}

func registerIPDiscovery(registry *registryruntime.Registry) {
	if registry == nil {
		return
	}
	_ = registry.RegisterResource(registryruntime.ResourceDescriptor{
		Group:    "network",
		Resource: "ip",
		Kind:     "network.ip",
		Verbs:    []string{"get", "list", "create", "update", "delete", "execute", "*"},
		DiscoverFunc: func(ctx context.Context, prefix string, _ string, _ int) (*metav1.List[discoverymodel.LookupItem], error) {
			groupsRes, err := iprepo.ScanAllPools(ctx)
			if err != nil {
				return nil, err
			}
			items := make([]discoverymodel.LookupItem, 0, len(groupsRes))
			for _, g := range groupsRes {
				if prefix == "" || strings.HasPrefix(g.ID, prefix) || strings.HasPrefix(g.Meta.Name, prefix) {
					items = append(items, discoverymodel.LookupItem{ID: g.ID, Name: g.Meta.Name})
				}
			}
			return &metav1.List[discoverymodel.LookupItem]{Items: items}, nil
		},
	})

	_ = registry.RegisterLookup("network/ip/pools", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		groupsRes, err := iprepo.ScanPools(ctx, "", 1000, search)
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		hasGlobal := perms.IsAllowed("network/ip")
		items := make([]discoverymodel.LookupItem, 0, len(groupsRes.Items))
		for _, g := range groupsRes.Items {
			if hasGlobal || perms.IsAllowed("network/ip/"+g.ID) {
				items = append(items, discoverymodel.LookupItem{
					ID:          g.ID,
					Name:        g.Meta.Name,
					Description: g.Meta.Description,
				})
			}
		}
		return registryruntime.Paginate(items, cursor, limit), nil
	})
}

func registerSiteDiscovery(registry *registryruntime.Registry) {
	if registry == nil {
		return
	}
	_ = registry.RegisterResource(registryruntime.ResourceDescriptor{
		Group:    "network",
		Resource: "site",
		Kind:     "network.site",
		Verbs:    []string{"get", "list", "create", "update", "delete", "execute", "*"},
		DiscoverFunc: func(ctx context.Context, prefix string, _ string, _ int) (*metav1.List[discoverymodel.LookupItem], error) {
			resp, err := siterepo.ScanGroups(ctx, "", 1000, "")
			if err != nil {
				return nil, err
			}
			items := make([]discoverymodel.LookupItem, 0, len(resp.Items))
			for _, g := range resp.Items {
				if prefix == "" || strings.HasPrefix(g.ID, prefix) || strings.HasPrefix(g.Meta.Name, prefix) {
					items = append(items, discoverymodel.LookupItem{ID: g.ID, Name: g.Meta.Name})
				}
			}
			return &metav1.List[discoverymodel.LookupItem]{Items: items}, nil
		},
	})

	_ = registry.RegisterLookup("network/site/pools", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		resp, err := siterepo.ScanGroups(ctx, "", 1000, search)
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		hasGlobal := perms.IsAllowed("network/site")
		items := make([]discoverymodel.LookupItem, 0, len(resp.Items))
		for _, g := range resp.Items {
			if hasGlobal || perms.IsAllowed("network/site/"+g.ID) {
				items = append(items, discoverymodel.LookupItem{
					ID:          g.ID,
					Name:        g.Meta.Name,
					Description: g.Meta.Description,
				})
			}
		}
		return registryruntime.Paginate(items, cursor, limit), nil
	})
}
