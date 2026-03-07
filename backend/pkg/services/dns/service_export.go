package dns

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsrepo "homelab/pkg/repositories/dns"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

var (
	exportCache *lru.Cache[string, exportCacheEntry]
)

type exportCacheEntry struct {
	Response     *models.DnsExportResponse
	LastModified time.Time
}

func init() {
	exportCache, _ = lru.New[string, exportCacheEntry](128)
}

func ClearCache() {
	exportCache.Purge()
}

func ExportAll(ctx context.Context) (*models.DnsExportResponse, error) {
	perms := commonauth.PermissionsFromContext(ctx)
	// Entry check: Allow if user has global 'dns' permission OR has specific instance permissions
	if !perms.AllowedAll && !perms.IsAllowed("network/dns") && len(perms.AllowedInstances) == 0 {
		return nil, fmt.Errorf("%w: dns", commonauth.ErrPermissionDenied)
	}

	domains, _, _ := dnsrepo.ListDomains(ctx, 0, 10000, "")
	all, _, _ := dnsrepo.ListRecords(ctx, "", 0, 100000, "")
	domainMap := make(map[string]map[string]map[string]interface{})
	for _, r := range all {
		if !r.Enabled {
			continue
		}
		if domainMap[r.DomainID] == nil {
			domainMap[r.DomainID] = make(map[string]map[string]interface{})
		}
		if domainMap[r.DomainID][r.Name] == nil {
			domainMap[r.DomainID][r.Name] = make(map[string]interface{})
		}
		domainMap[r.DomainID][r.Name][r.Type] = r.Value
	}
	resp := &models.DnsExportResponse{Domains: make([]models.ExportDomain, 0)}
	for _, d := range domains {
		if d.Enabled && perms.IsAllowed("network/dns/"+d.Name) {
			resp.Domains = append(resp.Domains, models.ExportDomain{Name: d.Name, Records: domainMap[d.ID]})
		}
	}
	return resp, nil
}
