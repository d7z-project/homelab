package rules

import (
	"context"
	"strings"
	"sync"

	metav1 "homelab/pkg/apis/meta/v1"
	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	ipmodel "homelab/pkg/models/network/ip"
	"homelab/pkg/models/shared"
	iprepo "homelab/pkg/repositories/network/ip"
	siterepo "homelab/pkg/repositories/network/site"
	registryruntime "homelab/pkg/runtime/registry"
)

var (
	registerDiscoveryOnce sync.Once
	registerIPOnce        sync.Once
	registerSiteOnce      sync.Once
)

func RegisterDiscovery() {
	registerDiscoveryOnce.Do(func() {
		RegisterIPDiscovery()
		RegisterSiteDiscovery()
	})
}

func RegisterIPDiscovery() {
	registerIPOnce.Do(registerIPDiscovery)
}

func RegisterSiteDiscovery() {
	registerSiteOnce.Do(registerSiteDiscovery)
}

func registerIPDiscovery() {
	_ = registryruntime.Default().RegisterResource(registryruntime.ResourceDescriptor{
		Group:    "network",
		Resource: "ip",
		Kind:     "network.ip",
		Verbs:    []string{"get", "list", "create", "update", "delete", "execute", "*"},
		DiscoverFunc: func(ctx context.Context, prefix string, _ string, _ int) (*metav1.List[discoverymodel.LookupItem], error) {
			groupsRes, err := iprepo.PoolRepo.List(ctx, "", 1000, nil)
			if err != nil {
				return nil, err
			}
			items := make([]discoverymodel.LookupItem, 0, len(groupsRes.Items))
			for _, g := range groupsRes.Items {
				if prefix == "" || strings.HasPrefix(g.ID, prefix) || strings.HasPrefix(g.Meta.Name, prefix) {
					items = append(items, discoverymodel.LookupItem{ID: g.ID, Name: g.Meta.Name})
				}
			}
			return &metav1.List[discoverymodel.LookupItem]{Items: items}, nil
		},
	})

	_ = registryruntime.Default().RegisterLookup("network/ip/pools", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		groupsRes, err := iprepo.PoolRepo.List(ctx, "", 1000, func(p *ipmodel.IPPool) bool {
			return search == "" || strings.Contains(strings.ToLower(p.Meta.Name), strings.ToLower(search)) || strings.Contains(strings.ToLower(p.ID), strings.ToLower(search))
		})
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

func registerSiteDiscovery() {
	_ = registryruntime.Default().RegisterResource(registryruntime.ResourceDescriptor{
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

	_ = registryruntime.Default().RegisterLookup("network/site/pools", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
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
