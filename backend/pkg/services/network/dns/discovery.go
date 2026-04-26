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
				domainID := "domain/" + normalizeDNSDomainResourcePart(d.Meta.Name)
				if prefix == "" || strings.HasPrefix(domainID, strings.ToLower(prefix)) {
					items = append(items, discoverymodel.LookupItem{ID: domainID, Name: d.Meta.Name})
				}

				recordPrefix := domainID + "/record/name/"
				if prefix != "" && (prefix == domainID || strings.HasPrefix(strings.ToLower(prefix), recordPrefix)) {
					namePrefix := ""
					if strings.HasPrefix(strings.ToLower(prefix), recordPrefix) {
						namePrefix = prefix[len(recordPrefix):]
					}

					recordsResp, err := dnsrepo.ScanRecords(ctx, d.ID, "", 10000, "")
					if err == nil {
						seen := make(map[string]bool)
						for _, r := range recordsResp.Items {
							key := dnsRecordResource(d.Meta.Name, r.Meta.Name, r.Meta.Type)
							if strings.HasPrefix(strings.ToLower(key), strings.ToLower(dnsResourceBase()+"/"+prefix)) && !seen[key] {
								seen[key] = true
								items = append(items, discoverymodel.LookupItem{
									ID:   strings.TrimPrefix(key, dnsResourceBase()+"/"),
									Name: r.Meta.Name + " (" + r.Meta.Type + ")",
								})
								continue
							}
							if namePrefix == "" && !seen[key] {
								seen[key] = true
								items = append(items, discoverymodel.LookupItem{
									ID:   strings.TrimPrefix(key, dnsResourceBase()+"/"),
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
			if perms.IsAllowed(dnsDomainResource(d.Meta.Name)) {
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
			resourceDomain := dnsDomainResource(domain.Meta.Name)
			resourceRecord := dnsRecordResource(domain.Meta.Name, r.Meta.Name, r.Meta.Type)
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
