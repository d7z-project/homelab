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

func ScanDomains(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.Domain], error) {
	resp, err := dnsrepo.DomainRepo.List(ctx, cursor, limit, nil)
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	if perms.IsAllowed("network/dns") {
		return resp, nil
	}

	// Filter by instance permissions
	var filtered []models.Domain
	for _, d := range resp.Items {
		if perms.IsAllowed("network/dns/" + d.Meta.Name) {
			filtered = append(filtered, d)
		}
	}
	resp.Items = filtered
	return resp, nil
}

func GetDomain(ctx context.Context, id string) (*models.Domain, error) {
	return dnsrepo.DomainRepo.Get(ctx, id)
}

func CreateDomain(ctx context.Context, domain *models.Domain) (*models.Domain, error) {
	if err := domain.Bind(nil); err != nil {
		return nil, err
	}
	resource := "network/dns/" + domain.Meta.Name
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) && !commonauth.PermissionsFromContext(ctx).IsAllowed("network/dns") {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	existingResp, _ := dnsrepo.DomainRepo.List(ctx, "", 1000, nil)
	if existingResp != nil {
		for _, ed := range existingResp.Items {
			if strings.EqualFold(ed.Meta.Name, domain.Meta.Name) {
				return nil, errors.New("domain already exists")
			}
		}
	}

	domain.ID = uuid.New().String()
	err := dnsrepo.DomainRepo.Cow(ctx, domain.ID, func(res *models.Resource[models.DomainV1Meta, models.DomainV1Status]) error {
		res.Meta = domain.Meta
		res.Status.CreatedAt = time.Now()
		res.Status.UpdatedAt = time.Now()
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	})

	if err != nil {
		commonaudit.FromContext(ctx).Log("CreateDomain", domain.Meta.Name, "Failed: "+err.Error(), false)
		return nil, err
	}

	defaultSOA := &models.Record{
		ID: uuid.New().String(),
		Meta: models.RecordV1Meta{
			DomainID: domain.ID,
			Name:     "@",
			Type:     "SOA",
			Value:    fmt.Sprintf("ns1.%s. admin.%s. %s %d %d %d %d", domain.Meta.Name, domain.Meta.Name, generateSOASerial(), defaultSOARefresh, defaultSOARetry, defaultSOAExpire, defaultSOAMinimum),
			TTL:      3600,
			Enabled:  true,
			Comments: "System generated SOA",
		},
	}
	_ = dnsrepo.RecordRepo.Cow(ctx, defaultSOA.ID, func(res *models.Resource[models.RecordV1Meta, models.RecordV1Status]) error {
		res.Meta = defaultSOA.Meta
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	})

	commonaudit.FromContext(ctx).Log("CreateDomain", domain.Meta.Name, "Created", true)
	updated, _ := dnsrepo.DomainRepo.Get(ctx, domain.ID)
	return updated, nil
}

func UpdateDomain(ctx context.Context, id string, domain *models.Domain) (*models.Domain, error) {
	if err := domain.Bind(nil); err != nil {
		return nil, err
	}
	existing, err := dnsrepo.DomainRepo.Get(ctx, id)
	if err != nil {
		return nil, errors.New("not found")
	}
	resource := "network/dns/" + existing.Meta.Name
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	err = dnsrepo.DomainRepo.PatchMeta(ctx, id, domain.Generation, func(m *models.DomainV1Meta) {
		// Only description is really editable for domain itself, name is immutable
		m.Description = domain.Meta.Description
	})

	if err != nil {
		commonaudit.FromContext(ctx).Log("UpdateDomain", existing.Meta.Name, "Failed: "+err.Error(), false)
		return nil, err
	}

	updated, _ := dnsrepo.DomainRepo.Get(ctx, id)
	commonaudit.FromContext(ctx).Log("UpdateDomain", existing.Meta.Name, "Updated", true)
	return updated, nil
}

func DeleteDomain(ctx context.Context, id string) error {
	existing, err := dnsrepo.DomainRepo.Get(ctx, id)
	if err != nil {
		return errors.New("not found")
	}
	resource := "network/dns/" + existing.Meta.Name
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	_ = dnsrepo.DeleteRecordsByDomain(ctx, id)
	err = dnsrepo.DomainRepo.Delete(ctx, id)
	commonaudit.FromContext(ctx).Log("DeleteDomain", existing.Meta.Name, "Deleted", err == nil)
	return err
}
