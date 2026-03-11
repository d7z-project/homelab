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
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/dns") {
		return nil, fmt.Errorf("%w: network/dns", commonauth.ErrPermissionDenied)
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

	record.ID = id
	record.Meta.DomainID = existing.Meta.DomainID
	if existing.Meta.Type == "SOA" {
		if record.Meta.Type != "SOA" || record.Meta.Name != "@" {
			return nil, errors.New("invalid SOA update")
		}
		m, r, _, err := parseSOA(record.Meta.Value)
		if err != nil {
			return nil, err
		}
		record.Meta.Value = fmt.Sprintf("%s %s %s %d %d %d %d", m, r, incrementSerial(existing.Meta.Value), defaultSOARefresh, defaultSOARetry, defaultSOAExpire, defaultSOAMinimum)
		record.Meta.Enabled = true
	}
	if err := validateRecord(ctx, record); err != nil {
		return nil, err
	}
	if err := dnsrepo.RecordRepo.Cow(ctx, record.ID, func(res *models.Record) error { res.Meta = record.Meta; res.Status = record.Status; return nil }); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRecord", record.Meta.Name+"."+dom.Meta.Name, "Failed", false)
		return nil, err
	}
	if existing.Meta.Type != "SOA" {
		updateSOASerial(ctx, dom.ID)
	}
	commonaudit.FromContext(ctx).Log("UpdateRecord", record.Meta.Name+"."+dom.Meta.Name, "Updated", true)
	return record, nil
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
