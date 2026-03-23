package dns

import (
	"context"
	"errors"
	"fmt"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsrepo "homelab/pkg/repositories/dns"
	"net"
	"strings"

	"github.com/google/uuid"
)

func ScanRecords(ctx context.Context, domainID string, cursor string, limit int, search string) (*models.PaginationResponse[models.Record], error) {
	if domainID == "" {
		perms := commonauth.PermissionsFromContext(ctx)
		if perms.IsAllowed("network/dns") {
			return dnsrepo.ScanRecords(ctx, "", cursor, limit, search)
		}
		return &models.PaginationResponse[models.Record]{
			Items: []models.Record{},
		}, nil
	}

	dom, err := dnsrepo.DomainRepo.Get(ctx, domainID)
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	if !perms.IsAllowed("network/dns") && !perms.IsAllowed("network/dns/"+dom.Meta.Name) {
		return nil, fmt.Errorf("%w: network/dns/%s", commonauth.ErrPermissionDenied, dom.Meta.Name)
	}
	return dnsrepo.ScanRecords(ctx, domainID, cursor, limit, search)
}

func GetRecord(ctx context.Context, id string) (*models.Record, error) {
	return dnsrepo.RecordRepo.Get(ctx, id)
}

func CreateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	if err := record.Bind(nil); err != nil {
		return nil, err
	}
	dom, err := dnsrepo.DomainRepo.Get(ctx, record.Meta.DomainID)
	if err != nil {
		return nil, errors.New("domain not found")
	}

	release, err := lockDomain(ctx, dom.ID)
	if err != nil {
		return nil, err
	}
	defer release()

	resource := fmt.Sprintf("network/dns/%s/%s/%s", dom.Meta.Name, record.Meta.Name, record.Meta.Type)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	if record.Meta.Type == "SOA" {
		return nil, errors.New("SOA managed by system")
	}
	if err := validateRecord(ctx, record); err != nil {
		return nil, err
	}

	record.ID = uuid.New().String()
	if err := dnsrepo.RecordRepo.Cow(ctx, record.ID, func(res *models.Record) error { res.Meta = record.Meta; res.Status = record.Status; return nil }); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRecord", record.Meta.Name+"."+dom.Meta.Name, "Failed", false)
		return nil, err
	}
	updateSOASerial(ctx, dom.ID)
	commonaudit.FromContext(ctx).Log("CreateRecord", record.Meta.Name+"."+dom.Meta.Name, "Created", true)
	return record, nil
}

func UpdateRecord(ctx context.Context, id string, record *models.Record) (*models.Record, error) {
	if err := record.Bind(nil); err != nil {
		return nil, err
	}
	existing, err := dnsrepo.RecordRepo.Get(ctx, id)
	if err != nil {
		return nil, errors.New("not found")
	}
	dom, _ := dnsrepo.DomainRepo.Get(ctx, existing.Meta.DomainID)
	if dom == nil {
		return nil, errors.New("domain not found")
	}

	release, err := lockDomain(ctx, dom.ID)
	if err != nil {
		return nil, err
	}
	defer release()

	resOld := fmt.Sprintf("network/dns/%s/%s/%s", dom.Meta.Name, existing.Meta.Name, existing.Meta.Type)
	resNew := fmt.Sprintf("network/dns/%s/%s/%s", dom.Meta.Name, record.Meta.Name, record.Meta.Type)
	perms := commonauth.PermissionsFromContext(ctx)
	if !perms.IsAllowed(resOld) || !perms.IsAllowed(resNew) {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resNew)
	}

	if err := validateRecord(ctx, record); err != nil {
		return nil, err
	}

	err = dnsrepo.RecordRepo.PatchMeta(ctx, id, record.Generation, func(meta *models.RecordV1Meta) {
		meta.Name = record.Meta.Name
		meta.Type = record.Meta.Type
		meta.Value = record.Meta.Value
		meta.TTL = record.Meta.TTL
		meta.Enabled = record.Meta.Enabled
		meta.Comments = record.Meta.Comments

		if existing.Meta.Type == "SOA" {
			// SOA specific logic handled inside patch to ensure atomicity and correct versioning
			meta.Name = "@"
			meta.Type = "SOA"
			meta.Enabled = true
			if mName, rName, _, err := parseSOA(record.Meta.Value); err == nil {
				meta.Value = fmt.Sprintf("%s %s %s %d %d %d %d", mName, rName, incrementSerial(existing.Meta.Value), defaultSOARefresh, defaultSOARetry, defaultSOAExpire, defaultSOAMinimum)
			}
		}
	})

	if err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRecord", record.Meta.Name+"."+dom.Meta.Name, "Failed", false)
		return nil, err
	}

	if existing.Meta.Type != "SOA" {
		updateSOASerial(ctx, dom.ID)
	}

	updated, _ := dnsrepo.RecordRepo.Get(ctx, id)
	commonaudit.FromContext(ctx).Log("UpdateRecord", record.Meta.Name+"."+dom.Meta.Name, "Updated", true)
	return updated, nil
}

func DeleteRecord(ctx context.Context, id string) error {
	existing, err := dnsrepo.RecordRepo.Get(ctx, id)
	if err != nil {
		return errors.New("not found")
	}
	dom, _ := dnsrepo.DomainRepo.Get(ctx, existing.Meta.DomainID)
	if dom == nil {
		return errors.New("domain not found")
	}

	release, err := lockDomain(ctx, dom.ID)
	if err != nil {
		return err
	}
	defer release()

	resource := fmt.Sprintf("network/dns/%s/%s/%s", dom.Meta.Name, existing.Meta.Name, existing.Meta.Type)
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	if existing.Meta.Type == "SOA" {
		return errors.New("cannot delete SOA")
	}
	err = dnsrepo.RecordRepo.Delete(ctx, id)
	if err == nil {
		updateSOASerial(ctx, dom.ID)
		commonaudit.FromContext(ctx).Log("DeleteRecord", existing.Meta.Name+"."+dom.Meta.Name, "Deleted", true)
	}
	return err
}

func validateRecord(ctx context.Context, record *models.Record) error {
	if record.Meta.Type == "A" && (net.ParseIP(record.Meta.Value) == nil || strings.Contains(record.Meta.Value, ":")) {
		return errors.New("invalid IPv4")
	}
	if record.Meta.Type == "AAAA" && (net.ParseIP(record.Meta.Value) == nil || !strings.Contains(record.Meta.Value, ":")) {
		return errors.New("invalid IPv6")
	}
	return nil
}
