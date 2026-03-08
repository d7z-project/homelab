package dns

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsrepo "homelab/pkg/repositories/dns"
	"homelab/pkg/services/discovery"
	"strings"
)

func init() {
	discovery.RegisterResourceWithVerbs("network/dns", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		res := make([]models.DiscoverResult, 0)
		resp, err := dnsrepo.ScanDomains(ctx, "", 10000, "")
		if err != nil {
			return nil, err
		}

		for _, d := range resp.Items {
			// 如果 prefix 为空或正在匹配域名本身
			if prefix == "" || strings.HasPrefix(d.Name, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: d.Name,
					Name:   d.Name,
					Final:  false, // 可以进一步探索记录
				})
			}

			// 如果 prefix 已经精确匹配了域名或其子路径
			if prefix != "" && (d.Name == prefix || strings.HasPrefix(prefix, d.Name+"/")) {
				idPrefix := ""
				if strings.HasPrefix(prefix, d.Name+"/") {
					idPrefix = strings.TrimPrefix(prefix, d.Name+"/")
				}

				recordsResp, err := dnsrepo.ScanRecords(ctx, d.ID, "", 10000, "")
				if err == nil {
					seen := make(map[string]bool)
					for _, r := range recordsResp.Items {
						// 记录发现：FullID = domain/record/type
						if strings.HasPrefix(r.Name, idPrefix) && !seen[r.Name+"/"+r.Type] {
							seen[r.Name+"/"+r.Type] = true
							res = append(res, models.DiscoverResult{
								FullID: d.Name + "/" + r.Name + "/" + r.Type,
								Name:   r.Name + " (" + r.Type + ")",
								Final:  true,
							})
						}
					}
				}
			}
		}
		return res, nil
	}, []string{"get", "list", "create", "update", "delete", "*"})

	discovery.Register("network/dns/domains", func(ctx context.Context, search string, cursor string, limit int) (*models.PaginationResponse[models.LookupItem], error) {
		resp, err := dnsrepo.ScanDomains(ctx, "", 10000, search)
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		var items []models.LookupItem
		for _, d := range resp.Items {
			if perms.IsAllowed("network/dns/" + d.Name) {
				items = append(items, models.LookupItem{
					ID:          d.ID,
					Name:        d.Name,
					Description: d.Comments,
				})
			}
		}
		return discovery.Paginate(items, cursor, limit), nil
	})

	discovery.Register("network/dns/records", func(ctx context.Context, search string, cursor string, limit int) (*models.PaginationResponse[models.LookupItem], error) {
		resp, err := dnsrepo.ScanRecords(ctx, "", "", 10000, search)
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		domainCache := make(map[string]*models.Domain)
		var items []models.LookupItem
		for _, r := range resp.Items {
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
		return discovery.Paginate(items, cursor, limit), nil
	})
}
