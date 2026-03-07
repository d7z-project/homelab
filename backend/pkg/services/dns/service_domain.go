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
	"strings"
	"time"

	"github.com/google/uuid"
)

func lockDomain(ctx context.Context, id string) (func(), error) {
	return common.LockWithTimeout(ctx, "network:dns:domain:"+id, 0)
}

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
