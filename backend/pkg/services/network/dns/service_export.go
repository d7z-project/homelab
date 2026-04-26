package dns

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	dnsmodel "homelab/pkg/models/network/dns"
	dnsrepo "homelab/pkg/repositories/network/dns"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

var (
	exportCache *lru.Cache[string, exportCacheEntry]
)

type exportCacheEntry struct {
	Response     *dnsmodel.DnsExportResponse
	LastModified time.Time
}

func init() {
	exportCache, _ = lru.New[string, exportCacheEntry](128)
}

func ClearCache() {
	exportCache.Purge()
}

func ExportAll(ctx context.Context) (*dnsmodel.DnsExportResponse, error) {
	perms := commonauth.PermissionsFromContext(ctx)
	if !perms.AllowedAll && !perms.IsAllowed("network/dns") && len(perms.AllowedInstances) == 0 {
		return nil, fmt.Errorf("%w: dns", commonauth.ErrPermissionDenied)
	}

	domains, _ := dnsrepo.ScanAllDomains(ctx)
	records, _ := dnsrepo.ScanAllRecords(ctx)

	recordsByDomain := make(map[string][]dnsmodel.ExportRecord)
	for _, r := range records {
		if !r.Meta.Enabled {
			continue
		}
		recordsByDomain[r.Meta.DomainID] = append(recordsByDomain[r.Meta.DomainID], dnsmodel.ExportRecord{
			Name:     r.Meta.Name,
			Type:     r.Meta.Type,
			Value:    recordValue(&r),
			TTL:      r.Meta.TTL,
			Priority: r.Meta.Priority,
		})
	}
	resp := &dnsmodel.DnsExportResponse{Domains: make([]dnsmodel.ExportDomain, 0)}
	for _, d := range domains {
		if d.Meta.Enabled && perms.IsAllowed("network/dns/"+d.Meta.Name) {
			resp.Domains = append(resp.Domains, dnsmodel.ExportDomain{
				Name:    d.Meta.Name,
				Records: recordsByDomain[d.ID],
			})
		}
	}
	return resp, nil
}
