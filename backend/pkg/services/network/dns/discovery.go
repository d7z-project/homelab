package dns

import (
	"context"
	"fmt"
	"strings"

	metav1 "homelab/pkg/apis/meta/v1"
	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	dnsmodel "homelab/pkg/models/network/dns"
	"homelab/pkg/models/shared"
	dnsrepo "homelab/pkg/repositories/network/dns"
	registryruntime "homelab/pkg/runtime/registry"
)

func RegisterDiscovery(registry *registryruntime.Registry) {
	if registry == nil {
		return
	}
	_ = registry.RegisterResource(registryruntime.ResourceDescriptor{
		Group:    "network",
		Resource: "dns",
		Kind:     "network.dns",
		Verbs:    []string{"get", "list", "create", "update", "delete", "*"},
		DiscoverFunc: func(ctx context.Context, prefix string, _ string, _ int) (*metav1.List[discoverymodel.LookupItem], error) {
			items := make([]discoverymodel.LookupItem, 0)
			resp, err := dnsrepo.ScanDomains(ctx, "", 10000, "")
			if err != nil {
				return nil, err
			}

			for _, d := range resp.Items {
				if prefix == "" || strings.HasPrefix(d.Meta.Name, prefix) {
					items = append(items, discoverymodel.LookupItem{ID: d.Meta.Name, Name: d.Meta.Name})
				}

				if prefix != "" && (d.Meta.Name == prefix || strings.HasPrefix(prefix, d.Meta.Name+"/")) {
					idPrefix := ""
					if strings.HasPrefix(prefix, d.Meta.Name+"/") {
						idPrefix = strings.TrimPrefix(prefix, d.Meta.Name+"/")
					}

					recordsResp, err := dnsrepo.ScanRecords(ctx, d.ID, "", 10000, "")
					if err == nil {
						seen := make(map[string]bool)
						for _, r := range recordsResp.Items {
							key := r.Meta.Name + "/" + r.Meta.Type
							if strings.HasPrefix(r.Meta.Name, idPrefix) && !seen[key] {
								seen[key] = true
								items = append(items, discoverymodel.LookupItem{
									ID:   d.Meta.Name + "/" + r.Meta.Name + "/" + r.Meta.Type,
									Name: r.Meta.Name + " (" + r.Meta.Type + ")",
								})
							}
						}
					}
				}
			}
			return &metav1.List[discoverymodel.LookupItem]{Items: items}, nil
		},
	})

	_ = registry.RegisterLookup("network/dns/domains", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		resp, err := dnsrepo.ScanDomains(ctx, "", 10000, "")
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		items := make([]discoverymodel.LookupItem, 0, len(resp.Items))
		for _, d := range resp.Items {
			if perms.IsAllowed("network/dns/" + d.Meta.Name) {
				items = append(items, discoverymodel.LookupItem{ID: d.ID, Name: d.Meta.Name})
			}
		}
		return registryruntime.Paginate(items, cursor, limit), nil
	})

	_ = registry.RegisterLookup("network/dns/records", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		resp, err := dnsrepo.ScanRecords(ctx, "", "", 10000, search)
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		domainCache := make(map[string]*dnsmodel.Domain)
		items := make([]discoverymodel.LookupItem, 0, len(resp.Items))
		for _, r := range resp.Items {
			domain, ok := domainCache[r.Meta.DomainID]
			if !ok {
				domain, _ = dnsrepo.GetDomain(ctx, r.Meta.DomainID)
				domainCache[r.Meta.DomainID] = domain
			}
			if domain == nil {
				continue
			}
			resourceDomain := fmt.Sprintf("network/dns/%s", domain.Meta.Name)
			resourceRecord := fmt.Sprintf("network/dns/%s/%s/%s", domain.Meta.Name, r.Meta.Name, r.Meta.Type)
			if perms.IsAllowed(resourceDomain) || perms.IsAllowed(resourceRecord) {
				items = append(items, discoverymodel.LookupItem{
					ID:          r.ID,
					Name:        fmt.Sprintf("%s (%s) - %s", r.Meta.Name, r.Meta.Type, domain.Meta.Name),
					Description: recordValue(&r),
				})
			}
		}
		return registryruntime.Paginate(items, cursor, limit), nil
	})
}
