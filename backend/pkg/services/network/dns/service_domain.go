package dns

import (
	"context"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	dnsmodel "homelab/pkg/models/network/dns"
	"homelab/pkg/models/shared"
	dnsrepo "homelab/pkg/repositories/network/dns"
	"strings"
	"time"

	"github.com/google/uuid"
)

func defaultDomainEmail(domainName string) string {
	return "admin@" + domainName
}

func defaultPrimaryNS(domainName string) string {
	return fmt.Sprintf("ns1.%s.", domainName)
}

func soaRName(email string) string {
	email = strings.TrimSpace(email)
	if email == "" {
		return ""
	}
	return strings.Replace(email, "@", ".", 1) + "."
}

func lockDomain(ctx context.Context, id string) (func(), error) {
	return common.LockWithTimeout(ctx, "network:dns:domain:"+id, 0)
}

func ScanDomains(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[dnsmodel.Domain], error) {
	resp, err := dnsrepo.ScanDomains(ctx, cursor, limit, search)
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	if perms.IsAllowed("network/dns") {
		return resp, nil
	}

	// Filter by instance permissions
	var filtered []dnsmodel.Domain
	for _, d := range resp.Items {
		if perms.IsAllowed("network/dns/" + d.Meta.Name) {
			filtered = append(filtered, d)
		}
	}
	resp.Items = filtered
	return resp, nil
}

func GetDomain(ctx context.Context, id string) (*dnsmodel.Domain, error) {
	return dnsrepo.GetDomain(ctx, id)
}

func CreateDomain(ctx context.Context, domain *dnsmodel.Domain) (*dnsmodel.Domain, error) {
	if err := normalizeDomain(domain); err != nil {
		return nil, err
	}
	resource := "network/dns/" + domain.Meta.Name
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) && !commonauth.PermissionsFromContext(ctx).IsAllowed("network/dns") {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	existingResp, _ := dnsrepo.ScanDomains(ctx, "", 1000, "")
	if existingResp != nil {
		for _, ed := range existingResp.Items {
			if strings.EqualFold(ed.Meta.Name, domain.Meta.Name) {
				return nil, errors.New("domain already exists")
			}
		}
	}

	domain.ID = uuid.New().String()
	if domain.Meta.Email == "" {
		domain.Meta.Email = defaultDomainEmail(domain.Meta.Name)
	}
	if domain.Meta.PrimaryNS == "" {
		domain.Meta.PrimaryNS = defaultPrimaryNS(domain.Meta.Name)
	}
	domain.Status.CreatedAt = time.Now()
	domain.Status.UpdatedAt = time.Now()
	err := dnsrepo.SaveDomain(ctx, domain)

	if err != nil {
		commonaudit.FromContext(ctx).Log("CreateDomain", domain.Meta.Name, "Failed: "+err.Error(), false)
		return nil, err
	}

	defaultSOA := &dnsmodel.Record{
		ID: uuid.New().String(),
		Meta: dnsmodel.RecordV1Meta{
			DomainID: domain.ID,
			Name:     "@",
			Type:     "SOA",
			TTL:      3600,
			Enabled:  true,
			Comments: "System generated SOA",
		},
		Status: dnsmodel.RecordV1Status{
			SOA: &dnsmodel.SOAStatus{
				MName:   domain.Meta.PrimaryNS,
				RName:   soaRName(domain.Meta.Email),
				Serial:  generateSOASerial(),
				Refresh: defaultSOARefresh,
				Retry:   defaultSOARetry,
				Expire:  defaultSOAExpire,
				Minimum: defaultSOAMinimum,
			},
		},
	}
	_ = dnsrepo.SaveRecord(ctx, defaultSOA)

	commonaudit.FromContext(ctx).Log("CreateDomain", domain.Meta.Name, "Created", true)
	updated, _ := dnsrepo.GetDomain(ctx, domain.ID)
	return updated, nil
}

func UpdateDomain(ctx context.Context, id string, domain *dnsmodel.Domain) (*dnsmodel.Domain, error) {
	if err := normalizeDomain(domain); err != nil {
		return nil, err
	}
	existing, err := dnsrepo.GetDomain(ctx, id)
	if err != nil {
		return nil, errors.New("not found")
	}
	resource := "network/dns/" + existing.Meta.Name
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	if domain.Meta.Email == "" {
		domain.Meta.Email = defaultDomainEmail(existing.Meta.Name)
	}
	if domain.Meta.PrimaryNS == "" {
		domain.Meta.PrimaryNS = defaultPrimaryNS(existing.Meta.Name)
	}
	existing.Meta.Description = domain.Meta.Description
	existing.Meta.Email = domain.Meta.Email
	existing.Meta.PrimaryNS = domain.Meta.PrimaryNS
	existing.Status.UpdatedAt = time.Now()
	err = dnsrepo.SaveDomain(ctx, existing)

	if err != nil {
		commonaudit.FromContext(ctx).Log("UpdateDomain", existing.Meta.Name, "Failed: "+err.Error(), false)
		return nil, err
	}
	resp, _ := dnsrepo.ScanRecords(ctx, existing.ID, "", 100, "")
	if resp != nil {
		for _, record := range resp.Items {
			if record.Meta.Type != "SOA" || record.Status.SOA == nil {
				continue
			}
			record.Status.SOA.MName = existing.Meta.PrimaryNS
			record.Status.SOA.RName = soaRName(existing.Meta.Email)
			record.Status.SOA.Serial = incrementSOASerial(record.Status.SOA.Serial)
			_ = dnsrepo.SaveRecord(ctx, &record)
			break
		}
	}

	updated, _ := dnsrepo.GetDomain(ctx, id)
	commonaudit.FromContext(ctx).Log("UpdateDomain", existing.Meta.Name, "Updated", true)
	return updated, nil
}

func DeleteDomain(ctx context.Context, id string) error {
	existing, err := dnsrepo.GetDomain(ctx, id)
	if err != nil {
		return errors.New("not found")
	}
	resource := "network/dns/" + existing.Meta.Name
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	_ = dnsrepo.DeleteRecordsByDomain(ctx, id)
	err = dnsrepo.DeleteDomain(ctx, id)
	commonaudit.FromContext(ctx).Log("DeleteDomain", existing.Meta.Name, "Deleted", err == nil)
	return err
}
