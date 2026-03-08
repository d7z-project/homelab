package dns

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsrepo "homelab/pkg/repositories/dns"
	"homelab/pkg/services/discovery"
	"homelab/pkg/services/rbac"
	"strings"
)

func init() {
	rbac.RegisterResourceWithVerbs("network/dns", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		res := make([]models.DiscoverResult, 0)
		domains, _, err := dnsrepo.ListDomains(ctx, 0, 10000, "")
		if err != nil {
			return nil, err
		}

		for _, d := range domains {
			if strings.HasPrefix(d.Name, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: d.Name,
					Name:   d.Name,
					Final:  true,
				})
			}

			if d.Name == prefix || strings.HasPrefix(prefix, d.Name+"/") {
				idPrefix := ""
				if strings.HasPrefix(prefix, d.Name+"/") {
					idPrefix = strings.TrimPrefix(prefix, d.Name+"/")
				}

				records, _, err := dnsrepo.ListRecords(ctx, d.ID, 0, 10000, "")
				if err == nil {
					seen := make(map[string]bool)
					for _, r := range records {
						if strings.HasPrefix(r.Name, idPrefix) && !seen[r.Name+"/"+r.Type] {
							seen[r.Name+"/"+r.Type] = true
							res = append(res, models.DiscoverResult{
								FullID: d.Name + "/" + r.Name + "/" + r.Type,
								Name:   r.Type,
								Final:  true,
							})
						}
					}
				}
			}
		}
		return res, nil
	}, []string{"get", "list", "create", "update", "delete", "*"})

	discovery.Register("network/dns/domains", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		domains, _, err := dnsrepo.ListDomains(ctx, 0, 10000, search)
		if err != nil {
			return nil, 0, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		var items []models.LookupItem
		for _, d := range domains {
			if perms.IsAllowed("network/dns/" + d.Name) {
				items = append(items, models.LookupItem{
					ID:          d.ID,
					Name:        d.Name,
					Description: d.Comments,
				})
			}
		}
		result, total := discovery.Paginate(items, offset, limit)
		return result, total, nil
	})

	discovery.Register("network/dns/records", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		records, _, err := dnsrepo.ListRecords(ctx, "", 0, 10000, search)
		if err != nil {
			return nil, 0, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		domainCache := make(map[string]*models.Domain)
		var items []models.LookupItem
		for _, r := range records {
			domain, ok := domainCache[r.DomainID]
			if !ok {
				domain, _ = dnsrepo.GetDomain(ctx, r.DomainID)
				domainCache[r.DomainID] = domain
			}
			if domain == nil {
				continue
			}
			resourceDomain := fmt.Sprintf("network/dns/%s", domain.Name)
			resourceRecord := fmt.Sprintf("network/dns/%s/%s/%s", domain.Name, r.Name, r.Type)
			if perms.IsAllowed(resourceDomain) || perms.IsAllowed(resourceRecord) {
				items = append(items, models.LookupItem{
					ID:          r.ID,
					Name:        fmt.Sprintf("%s (%s) - %s", r.Name, r.Type, domain.Name),
					Description: r.Value,
				})
			}
		}
		result, total := discovery.Paginate(items, offset, limit)
		return result, total, nil
	})
}
