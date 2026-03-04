package dns

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsrepo "homelab/pkg/repositories/dns"
	"homelab/pkg/services/discovery"
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	lru "github.com/hashicorp/golang-lru/v2"
)

var (
	domainRegex = regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
	exportCache *lru.Cache[string, exportCacheEntry]
)

type exportCacheEntry struct {
	Response     *models.DnsExportResponse
	LastModified time.Time
}

func init() {
	exportCache, _ = lru.New[string, exportCacheEntry](128)

	discovery.Register("dns/domains", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		domains, _, err := dnsrepo.ListDomains(ctx, 0, 10000, search)
		if err != nil {
			return nil, 0, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		var items []models.LookupItem
		for _, d := range domains {
			if perms.IsAllowed("dns/" + d.Name) {
				items = append(items, models.LookupItem{
					ID:          d.ID,
					Name:        d.Name,
					Description: d.Comments,
				})
			}
		}
		total := len(items)
		if offset >= total {
			return []models.LookupItem{}, total, nil
		}
		end := offset + limit
		if end > total {
			end = total
		}
		return items[offset:end], total, nil
	})

	discovery.Register("dns/records", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		records, _, err := dnsrepo.ListRecords(ctx, "", 0, 10000, search)
		if err != nil {
			return nil, 0, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		domainCache := make(map[string]*models.Domain)
		var items []models.LookupItem
		search = strings.ToLower(search)
		for _, r := range records {
			domain, ok := domainCache[r.DomainID]
			if !ok {
				domain, _ = dnsrepo.GetDomain(ctx, r.DomainID)
				domainCache[r.DomainID] = domain
			}
			if domain == nil {
				continue
			}
			resourceDomain := fmt.Sprintf("dns/%s", domain.Name)
			resourceRecord := fmt.Sprintf("dns/%s/%s/%s", domain.Name, r.Name, r.Type)
			if perms.IsAllowed(resourceDomain) || perms.IsAllowed(resourceRecord) {
				// Search check (already done in ListRecords, but adding ID check explicitly if needed)
				items = append(items, models.LookupItem{
					ID:          r.ID,
					Name:        fmt.Sprintf("%s.%s (%s)", r.Name, domain.Name, r.Type),
					Description: r.Value,
				})
			}
		}
		total := len(items)
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

func ClearCache() {
	exportCache.Purge()
}

const (
	defaultSOARefresh = 7200
	defaultSOARetry   = 3600
	defaultSOAExpire  = 1209600
	defaultSOAMinimum = 3600
)

func generateSOASerial() string {
	return time.Now().Format("20060102") + "01"
}

func parseSOA(value string) (mname, rname, rest string, err error) {
	parts := strings.Fields(value)
	if len(parts) < 2 {
		return "", "", "", errors.New("invalid SOA value")
	}
	mname = parts[0]
	rname = parts[1]
	if len(parts) > 2 {
		rest = strings.Join(parts[2:], " ")
	}
	return mname, rname, rest, nil
}

func incrementSerial(currentValue string) string {
	parts := strings.Fields(currentValue)
	if len(parts) < 3 {
		return generateSOASerial()
	}
	currentSerial := parts[2]
	today := time.Now().Format("20060102")
	if strings.HasPrefix(currentSerial, today) {
		// Increment within same day
		serialNum := 0
		fmt.Sscanf(currentSerial[len(today):], "%d", &serialNum)
		serialNum++
		return fmt.Sprintf("%s%02d", today, serialNum)
	}
	return today + "01"
}

func GetLastModified() time.Time {
	return dnsrepo.GetLastModified()
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
		resource := fmt.Sprintf("dns/%s", d.Name)
		if perms.IsAllowed(resource) {
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

	return &common.PaginatedResponse{
		Items: items,
		Total: total,
		Page:  page,
	}, nil
}

func CreateDomain(ctx context.Context, domain *models.Domain) (*models.Domain, error) {
	if err := domain.Bind(nil); err != nil {
		return nil, err
	}
	// Structural validation is now in models.Domain.Bind

	// Permission check for creating domain: dns/<name>
	resource := fmt.Sprintf("dns/%s", domain.Name)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, errors.New("permission denied: " + resource)
	}

	// Check if domain already exists (exact match)
	existingDomains, _, _ := dnsrepo.ListDomains(ctx, 0, 1000, domain.Name)
	for _, ed := range existingDomains {
		if strings.EqualFold(ed.Name, domain.Name) {
			return nil, errors.New("domain already exists")
		}
	}

	domain.ID = uuid.New().String()
	domain.CreatedAt = time.Now()
	domain.UpdatedAt = time.Now()

	message := fmt.Sprintf("Created domain %s (enabled: %v, comments: '%s')", domain.Name, domain.Enabled, domain.Comments)
	if err := dnsrepo.SaveDomain(ctx, domain); err != nil {
		commonaudit.FromContext(ctx).Log("CreateDomain", domain.Name, message, false)
		return nil, err
	}

	// Generate default SOA record
	defaultSOA := &models.Record{
		ID:       uuid.New().String(),
		DomainID: domain.ID,
		Name:     "@",
		Type:     "SOA",
		Value:    fmt.Sprintf("ns1.%s. admin.%s. %s %d %d %d %d", domain.Name, domain.Name, generateSOASerial(), defaultSOARefresh, defaultSOARetry, defaultSOAExpire, defaultSOAMinimum),
		TTL:      3600,
		Enabled:  true,
		Comments: "System generated default SOA",
	}
	if err := dnsrepo.SaveRecord(ctx, defaultSOA); err != nil {
		// Log error but domain is already created
		commonaudit.FromContext(ctx).Log("CreateRecord (SOA)", defaultSOA.Name+"."+domain.Name, "Failed to create default SOA record: "+err.Error(), false)
	}

	commonaudit.FromContext(ctx).Log("CreateDomain", domain.Name, message, true)
	return domain, nil
}

func UpdateDomain(ctx context.Context, id string, domain *models.Domain) (*models.Domain, error) {
	if err := domain.Bind(nil); err != nil {
		return nil, err
	}
	existing, err := dnsrepo.GetDomain(ctx, id)
	if err != nil {
		return nil, errors.New("domain not found")
	}

	// Permission check: dns/<domain-name>
	resource := fmt.Sprintf("dns/%s", existing.Name)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, errors.New("permission denied: " + resource)
	}

	changes := []string{}
	if existing.Enabled != domain.Enabled {
		changes = append(changes, fmt.Sprintf("enabled: %v -> %v", existing.Enabled, domain.Enabled))
	}
	if existing.Comments != domain.Comments {
		changes = append(changes, fmt.Sprintf("comments: '%s' -> '%s'", existing.Comments, domain.Comments))
	}
	message := fmt.Sprintf("Updated domain %s", existing.Name)
	if len(changes) > 0 {
		message += ": " + strings.Join(changes, ", ")
	} else {
		message += " (no changes)"
	}

	domain.ID = id
	domain.Name = existing.Name // Cannot change name
	domain.CreatedAt = existing.CreatedAt
	domain.UpdatedAt = time.Now()

	if err := dnsrepo.SaveDomain(ctx, domain); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateDomain", existing.Name, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateDomain", existing.Name, message, true)
	return domain, nil
}

func DeleteDomain(ctx context.Context, id string) error {
	existing, err := dnsrepo.GetDomain(ctx, id)
	if err != nil {
		return errors.New("domain not found")
	}

	// Permission check: dns/<domain-name>
	resource := fmt.Sprintf("dns/%s", existing.Name)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return errors.New("permission denied: " + resource)
	}

	message := fmt.Sprintf("Deleted domain %s (enabled: %v, comments: '%s') and all its records", existing.Name, existing.Enabled, existing.Comments)

	// Delete associated records first
	if err := dnsrepo.DeleteRecordsByDomain(ctx, id); err != nil {
		return fmt.Errorf("failed to delete records: %w", err)
	}

	if err := dnsrepo.DeleteDomain(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteDomain", existing.Name, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteDomain", existing.Name, message, true)
	return nil
}

// Record Service

func ListRecords(ctx context.Context, domainID string, page, pageSize int, search string) (*common.PaginatedResponse, error) {
	records, _, err := dnsrepo.ListRecords(ctx, domainID, 0, 10000, search)
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	domainCache := make(map[string]*models.Domain)

	var filteredRecords []models.Record
	for _, r := range records {
		domain, ok := domainCache[r.DomainID]
		if !ok {
			domain, _ = dnsrepo.GetDomain(ctx, r.DomainID)
			domainCache[r.DomainID] = domain
		}
		if domain == nil {
			continue
		}

		// Check if user has permission for this specific record or its domain
		resourceDomain := fmt.Sprintf("dns/%s", domain.Name)
		resourceRecord := fmt.Sprintf("dns/%s/%s/%s", domain.Name, r.Name, r.Type)
		if perms.IsAllowed(resourceDomain) || perms.IsAllowed(resourceRecord) {
			filteredRecords = append(filteredRecords, r)
		}
	}

	total := len(filteredRecords)
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
		items = append(items, filteredRecords[i])
	}

	return &common.PaginatedResponse{
		Items: items,
		Total: total,
		Page:  page,
	}, nil
}

func CreateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	if err := record.Bind(nil); err != nil {
		return nil, err
	}
	domain, err := dnsrepo.GetDomain(ctx, record.DomainID)
	if err != nil {
		return nil, errors.New("domain not found")
	}

	// Permission check: dns/<domain>/<host>/<type>
	resource := fmt.Sprintf("dns/%s/%s/%s", domain.Name, record.Name, record.Type)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, errors.New("permission denied: " + resource)
	}

	// Prevent manual SOA creation (handled by system)
	if record.Type == "SOA" {
		return nil, errors.New("SOA records are managed by the system and cannot be created manually")
	}

	if err := validateRecord(ctx, record); err != nil {
		return nil, err
	}

	record.ID = uuid.New().String()

	message := fmt.Sprintf("Created record %s in %s: %s -> %s (TTL: %d, enabled: %v, priority: %d)",
		record.Type, domain.Name, record.Name, record.Value, record.TTL, record.Enabled, record.Priority)

	if err := dnsrepo.SaveRecord(ctx, record); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRecord", record.Name+"."+domain.Name, message, false)
		return nil, err
	}

	// Update SOA serial
	updateSOASerial(ctx, domain.ID)

	commonaudit.FromContext(ctx).Log("CreateRecord", record.Name+"."+domain.Name, message, true)
	return record, nil
}

func UpdateRecord(ctx context.Context, id string, record *models.Record) (*models.Record, error) {
	if err := record.Bind(nil); err != nil {
		return nil, err
	}
	existing, err := dnsrepo.GetRecord(ctx, id)
	if err != nil {
		return nil, errors.New("record not found")
	}

	domain, err := dnsrepo.GetDomain(ctx, existing.DomainID)
	if err != nil {
		return nil, errors.New("domain not found")
	}

	// Permission check: dns/<domain>/<host>/<type> (Check both existing and new if changed)
	resourceOld := fmt.Sprintf("dns/%s/%s/%s", domain.Name, existing.Name, existing.Type)
	resourceNew := fmt.Sprintf("dns/%s/%s/%s", domain.Name, record.Name, record.Type)
	perms := commonauth.PermissionsFromContext(ctx)
	if !perms.IsAllowed(resourceOld) || !perms.IsAllowed(resourceNew) {
		return nil, errors.New("permission denied for this record operation")
	}

	record.ID = id
	record.DomainID = existing.DomainID // Cannot change domain of a record

	// Special handling for SOA
	isSOAUpdate := false
	if existing.Type == "SOA" {
		isSOAUpdate = true
		if record.Type != "SOA" {
			return nil, errors.New("cannot change type of SOA record")
		}
		if record.Name != "@" {
			return nil, errors.New("SOA record name must be '@'")
		}

		mname, rname, _, err := parseSOA(record.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid SOA format: %w", err)
		}

		// Keep current serial but increment it
		record.Value = fmt.Sprintf("%s %s %s %d %d %d %d",
			mname, rname, incrementSerial(existing.Value),
			defaultSOARefresh, defaultSOARetry, defaultSOAExpire, defaultSOAMinimum)

		// SOA must always be enabled
		record.Enabled = true
	}

	if err := validateRecord(ctx, record); err != nil {
		return nil, err
	}

	changes := []string{}
	if existing.Name != record.Name {
		changes = append(changes, fmt.Sprintf("host: %s -> %s", existing.Name, record.Name))
	}
	if existing.Type != record.Type {
		changes = append(changes, fmt.Sprintf("type: %s -> %s", existing.Type, record.Type))
	}
	if existing.Value != record.Value {
		changes = append(changes, fmt.Sprintf("value: %s -> %s", existing.Value, record.Value))
	}
	if existing.TTL != record.TTL {
		changes = append(changes, fmt.Sprintf("ttl: %d -> %d", existing.TTL, record.TTL))
	}
	if existing.Enabled != record.Enabled {
		changes = append(changes, fmt.Sprintf("enabled: %v -> %v", existing.Enabled, record.Enabled))
	}
	if existing.Priority != record.Priority {
		changes = append(changes, fmt.Sprintf("priority: %d -> %d", existing.Priority, record.Priority))
	}
	message := fmt.Sprintf("Updated record %s in %s", existing.Name, domain.Name)
	if len(changes) > 0 {
		message += ": " + strings.Join(changes, ", ")
	} else {
		message += " (no changes)"
	}

	if err := dnsrepo.SaveRecord(ctx, record); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRecord", existing.Name+"."+domain.Name, message, false)
		return nil, err
	}

	// Update SOA serial if NOT an SOA update (to avoid double update or recursion if we had any)
	if !isSOAUpdate {
		updateSOASerial(ctx, domain.ID)
	}

	commonaudit.FromContext(ctx).Log("UpdateRecord", existing.Name+"."+domain.Name, message, true)
	return record, nil
}

func DeleteRecord(ctx context.Context, id string) error {
	existing, err := dnsrepo.GetRecord(ctx, id)
	if err != nil {
		return errors.New("record not found")
	}

	domain, err := dnsrepo.GetDomain(ctx, existing.DomainID)
	if err != nil {
		return errors.New("domain not found")
	}

	// Permission check: dns/<domain>/<host>/<type>
	resource := fmt.Sprintf("dns/%s/%s/%s", domain.Name, existing.Name, existing.Type)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return errors.New("permission denied: " + resource)
	}

	// Prevent SOA deletion
	if existing.Type == "SOA" {
		return errors.New("SOA records cannot be deleted")
	}

	message := fmt.Sprintf("Deleted record: %s/%s -> %s (TTL: %d, enabled: %v, priority: %d) in %s",
		existing.Name, existing.Type, existing.Value, existing.TTL, existing.Enabled, existing.Priority, domain.Name)

	if err := dnsrepo.DeleteRecord(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRecord", existing.Name+"."+domain.Name, message, false)
		return err
	}

	// Update SOA serial
	updateSOASerial(ctx, domain.ID)

	commonaudit.FromContext(ctx).Log("DeleteRecord", existing.Name+"."+domain.Name, message, true)
	return nil
}

func updateSOASerial(ctx context.Context, domainID string) {
	records, _, err := dnsrepo.ListRecords(ctx, domainID, 0, 100, "")
	if err != nil {
		return
	}
	for _, r := range records {
		if r.Type == "SOA" {
			r.Value = fmt.Sprintf("%s %s %s %d %d %d %d",
				getPart(r.Value, 0), getPart(r.Value, 1), incrementSerial(r.Value),
				defaultSOARefresh, defaultSOARetry, defaultSOAExpire, defaultSOAMinimum)
			_ = dnsrepo.SaveRecord(ctx, &r)
			break
		}
	}
}

func getPart(value string, index int) string {
	parts := strings.Fields(value)
	if index < len(parts) {
		return parts[index]
	}
	return ""
}

func ExportAll(ctx context.Context) (*models.DnsExportResponse, error) {
	perms := commonauth.PermissionsFromContext(ctx)
	// Use JSON marshaled permissions as a stable cache key
	permsData, _ := json.Marshal(perms)
	cacheKey := string(permsData)
	lastMod := dnsrepo.GetLastModified()

	if entry, ok := exportCache.Get(cacheKey); ok {
		if !entry.LastModified.Before(lastMod) {
			return entry.Response, nil
		}
	}

	// Fetch all domains
	domains, _, err := dnsrepo.ListDomains(ctx, 0, 1000, "")
	if err != nil {
		return nil, err
	}

	// Fetch all records
	allRecords, _, err := dnsrepo.ListRecords(ctx, "", 0, 10000, "")
	if err != nil {
		return nil, err
	}

	// Map enabled records to domain IDs: map[domainID]map[name]map[type]interface{}
	domainMap := make(map[string]map[string]map[string]interface{})
	for _, r := range allRecords {
		if !r.Enabled {
			continue
		}

		if _, ok := domainMap[r.DomainID]; !ok {
			domainMap[r.DomainID] = make(map[string]map[string]interface{})
		}
		if _, ok := domainMap[r.DomainID][r.Name]; !ok {
			domainMap[r.DomainID][r.Name] = make(map[string]interface{})
		}

		var exportRec interface{}
		switch r.Type {
		case "A":
			exportRec = models.ExportRecordA{Address: r.Value, TTL: r.TTL}
		case "AAAA":
			exportRec = models.ExportRecordAAAA{Address: r.Value, TTL: r.TTL}
		case "CNAME":
			exportRec = models.ExportRecordCNAME{Target: r.Value, TTL: r.TTL}
		case "NS":
			exportRec = models.ExportRecordNS{Target: r.Value, TTL: r.TTL}
		case "MX":
			exportRec = models.ExportRecordMX{Host: r.Value, Priority: r.Priority, TTL: r.TTL}
		case "TXT":
			exportRec = models.ExportRecordTXT{Text: r.Value, TTL: r.TTL}
		case "SRV":
			parts := strings.Fields(r.Value)
			weight := 0
			port := 0
			target := ""
			if len(parts) >= 3 {
				fmt.Sscanf(parts[0], "%d", &weight)
				fmt.Sscanf(parts[1], "%d", &port)
				target = strings.Join(parts[2:], " ")
			}
			exportRec = models.ExportRecordSRV{Priority: r.Priority, Weight: weight, Port: port, Target: target, TTL: r.TTL}
		case "CAA":
			parts := strings.Fields(r.Value)
			flags := 0
			tag := ""
			value := ""
			if len(parts) >= 3 {
				fmt.Sscanf(parts[0], "%d", &flags)
				tag = parts[1]
				value = strings.Join(parts[2:], " ")
				if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
					value = value[1 : len(value)-1]
				}
			}
			exportRec = models.ExportRecordCAA{Flags: flags, Tag: tag, Value: value, TTL: r.TTL}
		case "SOA":
			parts := strings.Fields(r.Value)
			var serial int64
			refresh, retry, expire, minimum := 0, 0, 0, 0
			mname, rname := "", ""
			if len(parts) >= 7 {
				mname = parts[0]
				rname = parts[1]
				fmt.Sscanf(parts[2], "%d", &serial)
				fmt.Sscanf(parts[3], "%d", &refresh)
				fmt.Sscanf(parts[4], "%d", &retry)
				fmt.Sscanf(parts[5], "%d", &expire)
				fmt.Sscanf(parts[6], "%d", &minimum)
			}
			exportRec = models.ExportRecordSOA{
				Mname: mname, Rname: rname, Serial: serial,
				Refresh: refresh, Retry: retry, Expire: expire, Minimum: minimum,
				TTL: r.TTL,
			}
		}

		if r.Type == "SOA" {
			// SOA is single
			domainMap[r.DomainID][r.Name][r.Type] = exportRec
		} else {
			// Others are arrays
			if _, ok := domainMap[r.DomainID][r.Name][r.Type]; !ok {
				domainMap[r.DomainID][r.Name][r.Type] = []interface{}{}
			}
			domainMap[r.DomainID][r.Name][r.Type] = append(domainMap[r.DomainID][r.Name][r.Type].([]interface{}), exportRec)
		}
	}

	// Construct response with only enabled domains and those allowed by permissions
	resp := &models.DnsExportResponse{
		Domains: make([]models.ExportDomain, 0),
	}

	for _, d := range domains {
		if !d.Enabled {
			continue
		}
		// Permission check: dns/<domain-name>
		resource := fmt.Sprintf("dns/%s", d.Name)
		if !perms.IsAllowed(resource) {
			continue
		}

		recordsForDomain := domainMap[d.ID]
		if recordsForDomain == nil {
			recordsForDomain = make(map[string]map[string]interface{})
		}

		exportDom := models.ExportDomain{
			Name:    d.Name,
			Records: recordsForDomain,
		}
		resp.Domains = append(resp.Domains, exportDom)
	}

	// Cache the result
	exportCache.Add(cacheKey, exportCacheEntry{
		Response:     resp,
		LastModified: lastMod,
	})

	return resp, nil
}

func validateRecord(ctx context.Context, record *models.Record) error {
	// Basic non-empty checks moved to models.Record.Bind

	// Validate Value based on Type
	switch record.Type {
	case "A":
		if net.ParseIP(record.Value) == nil || strings.Contains(record.Value, ":") {
			return errors.New("invalid IPv4 address")
		}
	case "AAAA":
		if net.ParseIP(record.Value) == nil || !strings.Contains(record.Value, ":") {
			return errors.New("invalid IPv6 address")
		}
	}

	// RFC 1034 CNAME mutual exclusion
	records, _, err := dnsrepo.ListRecords(ctx, record.DomainID, 0, 1000, "")
	if err != nil {
		return err
	}

	for _, r := range records {
		if r.ID == record.ID {
			continue
		}
		if r.Name == record.Name {
			if record.Type == "CNAME" {
				return errors.New("CNAME record cannot coexist with other records of the same name")
			}
			if r.Type == "CNAME" {
				return errors.New("cannot create record because a CNAME already exists for this name")
			}
		}
	}

	return nil
}
