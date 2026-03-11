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
		resp, err := dnsrepo.DomainRepo.List(ctx, "", 10000, nil)
		if err != nil {
			return nil, err
		}

		for _, d := range resp.Items {
			// 如果 prefix 为空或正在匹配域名本身
			if prefix == "" || strings.HasPrefix(d.Meta.Name, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: d.Meta.Name,
					Name:   d.Meta.Name,
					Final:  false, // 可以进一步探索记录
				})
			}

			// 如果 prefix 已经精确匹配了域名或其子路径
			if prefix != "" && (d.Meta.Name == prefix || strings.HasPrefix(prefix, d.Meta.Name+"/")) {
				idPrefix := ""
				if strings.HasPrefix(prefix, d.Meta.Name+"/") {
					idPrefix = strings.TrimPrefix(prefix, d.Meta.Name+"/")
				}

				recordsResp, err := dnsrepo.ScanRecords(ctx, d.ID, "", 10000, "")
				if err == nil {
					seen := make(map[string]bool)
					for _, r := range recordsResp.Items {
						// 记录发现：FullID = domain/record/type
						if strings.HasPrefix(r.Meta.Name, idPrefix) && !seen[r.Meta.Name+"/"+r.Meta.Type] {
							seen[r.Meta.Name+"/"+r.Meta.Type] = true
							res = append(res, models.DiscoverResult{
								FullID: d.Meta.Name + "/" + r.Meta.Name + "/" + r.Meta.Type,
								Name:   r.Meta.Name + " (" + r.Meta.Type + ")",
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
		resp, err := dnsrepo.DomainRepo.List(ctx, "", 10000, nil)
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		var items []models.LookupItem
		for _, d := range resp.Items {
			if perms.IsAllowed("network/dns/" + d.Meta.Name) {
				items = append(items, models.LookupItem{
					ID:          d.ID,
					Name:        d.Meta.Name,
					Description: "",
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
			domain, ok := domainCache[r.Meta.DomainID]
			if !ok {
				domain, _ = dnsrepo.DomainRepo.Get(ctx, r.Meta.DomainID)
				domainCache[r.Meta.DomainID] = domain
			}
			if domain == nil {
				continue
			}
			resourceDomain := fmt.Sprintf("network/dns/%s", domain.Meta.Name)
			resourceRecord := fmt.Sprintf("network/dns/%s/%s/%s", domain.Meta.Name, r.Meta.Name, r.Meta.Type)
			if perms.IsAllowed(resourceDomain) || perms.IsAllowed(resourceRecord) {
				items = append(items, models.LookupItem{
					ID:          r.ID,
					Name:        fmt.Sprintf("%s (%s) - %s", r.Meta.Name, r.Meta.Type, domain.Meta.Name),
					Description: r.Meta.Value,
				})
			}
		}
		return discovery.Paginate(items, cursor, limit), nil
	})
}
