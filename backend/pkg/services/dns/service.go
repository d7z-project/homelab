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
	"net"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	domainRegex = regexp.MustCompile(`^([a-z0-9]+(-[a-z0-9]+)*\.)+[a-z]{2,}$`)
)

// Domain Service

func ListDomains(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
	domains, total, err := dnsrepo.ListDomains(ctx, page-1, pageSize, search)
	if err != nil {
		return nil, err
	}

	var items []interface{}
	for _, d := range domains {
		items = append(items, d)
	}

	return &common.PaginatedResponse{
		Items: items,
		Total: total,
		Page:  page,
	}, nil
}

func CreateDomain(ctx context.Context, domain *models.Domain) (*models.Domain, error) {
	if domain.Name == "" {
		return nil, errors.New("domain name is required")
	}
	domain.Name = strings.ToLower(domain.Name)
	if !domainRegex.MatchString(domain.Name) {
		return nil, errors.New("invalid domain name format")
	}

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

	message := fmt.Sprintf("Created domain %s (enabled: %v)", domain.Name, domain.Enabled)
	if err := dnsrepo.SaveDomain(ctx, domain); err != nil {
		commonaudit.FromContext(ctx).Log("CreateDomain", domain.Name, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateDomain", domain.Name, message, true)
	return domain, nil
}

func UpdateDomain(ctx context.Context, id string, domain *models.Domain) (*models.Domain, error) {
	existing, err := dnsrepo.GetDomain(ctx, id)
	if err != nil {
		return nil, errors.New("domain not found")
	}

	// Permission check: dns/<domain-name>
	resource := fmt.Sprintf("dns/%s", existing.Name)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, errors.New("permission denied: " + resource)
	}

	message := fmt.Sprintf("Updated domain %s", existing.Name)
	changes := []string{}
	if existing.Enabled != domain.Enabled {
		changes = append(changes, fmt.Sprintf("enabled: %v -> %v", existing.Enabled, domain.Enabled))
	}
	if existing.Comments != domain.Comments {
		changes = append(changes, fmt.Sprintf("comments: '%s' -> '%s'", existing.Comments, domain.Comments))
	}
	if len(changes) > 0 {
		message += " (" + strings.Join(changes, ", ") + ")"
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

	message := fmt.Sprintf("Deleted domain %s and all its records", existing.Name)

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
	// If domainID is provided, check permission for that domain
	if domainID != "" {
		existing, err := dnsrepo.GetDomain(ctx, domainID)
		if err == nil {
			resource := fmt.Sprintf("dns/%s", existing.Name)
			if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
				return nil, errors.New("permission denied for domain " + existing.Name)
			}
		}
	}

	records, total, err := dnsrepo.ListRecords(ctx, domainID, page-1, pageSize, search)
	if err != nil {
		return nil, err
	}

	var items []interface{}
	for _, r := range records {
		items = append(items, r)
	}

	return &common.PaginatedResponse{
		Items: items,
		Total: total,
		Page:  page,
	}, nil
}

func CreateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	domain, err := dnsrepo.GetDomain(ctx, record.DomainID)
	if err != nil {
		return nil, errors.New("domain not found")
	}

	// Permission check: dns/<domain>/<host>/<type>
	resource := fmt.Sprintf("dns/%s/%s/%s", domain.Name, record.Name, record.Type)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, errors.New("permission denied: " + resource)
	}

	if err := validateRecord(ctx, record); err != nil {
		return nil, err
	}

	record.ID = uuid.New().String()

	message := fmt.Sprintf("Created record %s in %s: %s -> %s (TTL: %d, enabled: %v)",
		record.Type, domain.Name, record.Name, record.Value, record.TTL, record.Enabled)

	if err := dnsrepo.SaveRecord(ctx, record); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRecord", record.Name+"."+domain.Name, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateRecord", record.Name+"."+domain.Name, message, true)
	return record, nil
}

func UpdateRecord(ctx context.Context, id string, record *models.Record) (*models.Record, error) {
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

	if err := validateRecord(ctx, record); err != nil {
		return nil, err
	}

	message := fmt.Sprintf("Updated record %s in %s", existing.Name, domain.Name)
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
	if len(changes) > 0 {
		message += " (" + strings.Join(changes, ", ") + ")"
	}

	if err := dnsrepo.SaveRecord(ctx, record); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRecord", existing.Name+"."+domain.Name, message, false)
		return nil, err
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

	message := fmt.Sprintf("Deleted %s record: %s in %s", existing.Type, existing.Name, domain.Name)

	if err := dnsrepo.DeleteRecord(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRecord", existing.Name+"."+domain.Name, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteRecord", existing.Name+"."+domain.Name, message, true)
	return nil
}

func ExportAll(ctx context.Context) (*models.DnsExportResponse, error) {
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

	// Map enabled records to domain IDs
	recordMap := make(map[string][]models.ExportRecord)
	for _, r := range allRecords {
		if !r.Enabled {
			continue
		}
		exportRec := models.ExportRecord{
			Name:     r.Name,
			Type:     r.Type,
			Value:    r.Value,
			TTL:      r.TTL,
			Priority: r.Priority,
		}
		recordMap[r.DomainID] = append(recordMap[r.DomainID], exportRec)
	}

	// Construct response with only enabled domains
	resp := &models.DnsExportResponse{
		Domains: make([]models.ExportDomain, 0),
	}

	for _, d := range domains {
		if !d.Enabled {
			continue
		}
		exportDom := models.ExportDomain{
			Name:    d.Name,
			Records: recordMap[d.ID],
		}
		if exportDom.Records == nil {
			exportDom.Records = []models.ExportRecord{}
		}
		resp.Domains = append(resp.Domains, exportDom)
	}

	return resp, nil
}

func validateRecord(ctx context.Context, record *models.Record) error {
	if record.Name == "" {
		return errors.New("record name is required")
	}
	if record.Type == "" {
		return errors.New("record type is required")
	}
	if record.Value == "" {
		return errors.New("record value is required")
	}

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
