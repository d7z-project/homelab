package dns

import (
	"context"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsrepo "homelab/pkg/repositories/dns"
	"homelab/pkg/services/discovery"
	"homelab/pkg/services/rbac"
	"net"
	"strings"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
)

const (
	defaultSOARefresh = 7200
	defaultSOARetry   = 3600
	defaultSOAExpire  = 1209600
	defaultSOAMinimum = 86400
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
		total := len(items)
		if limit <= 0 {
			limit = 20
		}
		if offset >= total {
			return []models.LookupItem{}, total, nil
		}
		end := offset + limit
		if end > total {
			end = total
		}
		return items[offset:end], total, nil
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
		total := len(items)
		if limit <= 0 {
			limit = 20
		}
		if offset >= total {
			return []models.LookupItem{}, total, nil
		}
		end := offset + limit
		if end > total {
			end = total
		}
		return items[offset:end], total, nil
	})
}

// Domain Service

func ListDomains(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
	domains, _, err := dnsrepo.ListDomains(ctx, 0, 10000, search)
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	var filteredDomains []models.Domain
	for _, d := range domains {
		if perms.IsAllowed("network/dns") || perms.IsAllowed("network/dns/"+d.Name) {
			filteredDomains = append(filteredDomains, d)
		}
	}

	total := len(filteredDomains)
	start := (page - 1) * pageSize
	if start >= total {
		return &common.PaginatedResponse{Items: []interface{}{}, Total: total, Page: page}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	var items []interface{}
	for i := start; i < end; i++ {
		items = append(items, filteredDomains[i])
	}

	return &common.PaginatedResponse{Items: items, Total: total, Page: page}, nil
}

func CreateDomain(ctx context.Context, domain *models.Domain) (*models.Domain, error) {
	if err := domain.Bind(nil); err != nil {
		return nil, err
	}
	resource := "network/dns/" + domain.Name
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	existingDomains, _, _ := dnsrepo.ListDomains(ctx, 0, 1000, domain.Name)
	for _, ed := range existingDomains {
		if strings.EqualFold(ed.Name, domain.Name) {
			return nil, errors.New("domain already exists")
		}
	}

	domain.ID = uuid.New().String()
	domain.CreatedAt = time.Now()
	domain.UpdatedAt = time.Now()

	if err := dnsrepo.SaveDomain(ctx, domain); err != nil {
		commonaudit.FromContext(ctx).Log("CreateDomain", domain.Name, "Failed: "+err.Error(), false)
		return nil, err
	}

	defaultSOA := &models.Record{
		ID: uuid.New().String(), DomainID: domain.ID, Name: "@", Type: "SOA",
		Value: fmt.Sprintf("ns1.%s. admin.%s. %s %d %d %d %d", domain.Name, domain.Name, generateSOASerial(), defaultSOARefresh, defaultSOARetry, defaultSOAExpire, defaultSOAMinimum),
		TTL:   3600, Enabled: true, Comments: "System generated SOA",
	}
	_ = dnsrepo.SaveRecord(ctx, defaultSOA)

	commonaudit.FromContext(ctx).Log("CreateDomain", domain.Name, "Created", true)
	return domain, nil
}

func UpdateDomain(ctx context.Context, id string, domain *models.Domain) (*models.Domain, error) {
	if err := domain.Bind(nil); err != nil {
		return nil, err
	}
	existing, err := dnsrepo.GetDomain(ctx, id)
	if err != nil {
		return nil, errors.New("not found")
	}
	resource := "network/dns/" + existing.Name
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	domain.ID = id
	domain.Name = existing.Name
	domain.CreatedAt = existing.CreatedAt
	domain.UpdatedAt = time.Now()

	if err := dnsrepo.SaveDomain(ctx, domain); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateDomain", existing.Name, "Failed: "+err.Error(), false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateDomain", existing.Name, "Updated", true)
	return domain, nil
}

func DeleteDomain(ctx context.Context, id string) error {
	existing, err := dnsrepo.GetDomain(ctx, id)
	if err != nil {
		return errors.New("not found")
	}
	resource := "network/dns/" + existing.Name
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	_ = dnsrepo.DeleteRecordsByDomain(ctx, id)
	err = dnsrepo.DeleteDomain(ctx, id)
	commonaudit.FromContext(ctx).Log("DeleteDomain", existing.Name, "Deleted", err == nil)
	return err
}

// Record Service

func ListRecords(ctx context.Context, domainID string, page, pageSize int, search string) (*common.PaginatedResponse, error) {
	records, _, err := dnsrepo.ListRecords(ctx, domainID, 0, 10000, search)
	if err != nil {
		return nil, err
	}
	perms := commonauth.PermissionsFromContext(ctx)
	domainCache := make(map[string]*models.Domain)

	var filtered []models.Record
	for _, r := range records {
		dom, ok := domainCache[r.DomainID]
		if !ok {
			dom, _ = dnsrepo.GetDomain(ctx, r.DomainID)
			domainCache[r.DomainID] = dom
		}
		if dom != nil && (perms.IsAllowed("network/dns/"+dom.Name) || perms.IsAllowed("network/dns/"+dom.Name+"/"+r.Name+"/"+r.Type)) {
			filtered = append(filtered, r)
		}
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	if start >= total {
		return &common.PaginatedResponse{Items: []interface{}{}, Total: total, Page: page}, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	var items []interface{}
	for i := start; i < end; i++ {
		items = append(items, filtered[i])
	}
	return &common.PaginatedResponse{Items: items, Total: total, Page: page}, nil
}

func CreateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	if err := record.Bind(nil); err != nil {
		return nil, err
	}
	dom, err := dnsrepo.GetDomain(ctx, record.DomainID)
	if err != nil {
		return nil, errors.New("domain not found")
	}
	resource := fmt.Sprintf("network/dns/%s/%s/%s", dom.Name, record.Name, record.Type)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	if record.Type == "SOA" {
		return nil, errors.New("SOA managed by system")
	}
	if err := validateRecord(ctx, record); err != nil {
		return nil, err
	}

	record.ID = uuid.New().String()
	if err := dnsrepo.SaveRecord(ctx, record); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRecord", record.Name+"."+dom.Name, "Failed", false)
		return nil, err
	}
	updateSOASerial(ctx, dom.ID)
	commonaudit.FromContext(ctx).Log("CreateRecord", record.Name+"."+dom.Name, "Created", true)
	return record, nil
}

func UpdateRecord(ctx context.Context, id string, record *models.Record) (*models.Record, error) {
	if err := record.Bind(nil); err != nil {
		return nil, err
	}
	existing, err := dnsrepo.GetRecord(ctx, id)
	if err != nil {
		return nil, errors.New("not found")
	}
	dom, _ := dnsrepo.GetDomain(ctx, existing.DomainID)
	if dom == nil {
		return nil, errors.New("domain not found")
	}

	resOld := fmt.Sprintf("network/dns/%s/%s/%s", dom.Name, existing.Name, existing.Type)
	resNew := fmt.Sprintf("network/dns/%s/%s/%s", dom.Name, record.Name, record.Type)
	perms := commonauth.PermissionsFromContext(ctx)
	if !perms.IsAllowed(resOld) || !perms.IsAllowed(resNew) {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resNew)
	}

	record.ID = id
	record.DomainID = existing.DomainID
	if existing.Type == "SOA" {
		if record.Type != "SOA" || record.Name != "@" {
			return nil, errors.New("invalid SOA update")
		}
		m, r, _, err := parseSOA(record.Value)
		if err != nil {
			return nil, err
		}
		record.Value = fmt.Sprintf("%s %s %s %d %d %d %d", m, r, incrementSerial(existing.Value), defaultSOARefresh, defaultSOARetry, defaultSOAExpire, defaultSOAMinimum)
		record.Enabled = true
	}
	if err := validateRecord(ctx, record); err != nil {
		return nil, err
	}
	if err := dnsrepo.SaveRecord(ctx, record); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRecord", record.Name+"."+dom.Name, "Failed", false)
		return nil, err
	}
	if existing.Type != "SOA" {
		updateSOASerial(ctx, dom.ID)
	}
	commonaudit.FromContext(ctx).Log("UpdateRecord", record.Name+"."+dom.Name, "Updated", true)
	return record, nil
}

func DeleteRecord(ctx context.Context, id string) error {
	existing, err := dnsrepo.GetRecord(ctx, id)
	if err != nil {
		return errors.New("not found")
	}
	dom, _ := dnsrepo.GetDomain(ctx, existing.DomainID)
	if dom == nil {
		return errors.New("domain not found")
	}
	resource := fmt.Sprintf("network/dns/%s/%s/%s", dom.Name, existing.Name, existing.Type)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	if existing.Type == "SOA" {
		return errors.New("cannot delete SOA")
	}
	err = dnsrepo.DeleteRecord(ctx, id)
	if err == nil {
		updateSOASerial(ctx, dom.ID)
		commonaudit.FromContext(ctx).Log("DeleteRecord", existing.Name+"."+dom.Name, "Deleted", true)
	}
	return err
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

func generateSOASerial() string { return time.Now().Format("20060102") + "01" }
func updateSOASerial(ctx context.Context, domainID string) {
	records, _, _ := dnsrepo.ListRecords(ctx, domainID, 0, 100, "")
	for _, r := range records {
		if r.Type == "SOA" {
			m, rn, _, _ := parseSOA(r.Value)
			r.Value = fmt.Sprintf("%s %s %s %d %d %d %d", m, rn, incrementSerial(r.Value), defaultSOARefresh, defaultSOARetry, defaultSOAExpire, defaultSOAMinimum)
			_ = dnsrepo.SaveRecord(ctx, &r)
			break
		}
	}
}
func parseSOA(val string) (m, r, s string, err error) {
	p := strings.Fields(val)
	if len(p) < 3 {
		return "", "", "", errors.New("invalid SOA")
	}
	return p[0], p[1], p[2], nil
}
func incrementSerial(old string) string {
	_, _, s, err := parseSOA(old)
	today := time.Now().Format("20060102")
	if err != nil || !strings.HasPrefix(s, today) {
		return today + "01"
	}
	seq := 1
	fmt.Sscanf(s[8:], "%d", &seq)
	return today + fmt.Sprintf("%02d", seq+1)
}
func validateRecord(ctx context.Context, record *models.Record) error {
	if record.Type == "A" && (net.ParseIP(record.Value) == nil || strings.Contains(record.Value, ":")) {
		return errors.New("invalid IPv4")
	}
	if record.Type == "AAAA" && (net.ParseIP(record.Value) == nil || !strings.Contains(record.Value, ":")) {
		return errors.New("invalid IPv6")
	}
	return nil
}
