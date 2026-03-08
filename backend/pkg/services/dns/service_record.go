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

func CreateRecord(ctx context.Context, record *models.Record) (*models.Record, error) {
	if err := record.Bind(nil); err != nil {
		return nil, err
	}
	dom, err := dnsrepo.GetDomain(ctx, record.DomainID)
	if err != nil {
		return nil, errors.New("domain not found")
	}

	release, err := lockDomain(ctx, dom.ID)
	if err != nil {
		return nil, err
	}
	defer release()

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

	release, err := lockDomain(ctx, dom.ID)
	if err != nil {
		return nil, err
	}
	defer release()

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

	release, err := lockDomain(ctx, dom.ID)
	if err != nil {
		return err
	}
	defer release()

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

func validateRecord(ctx context.Context, record *models.Record) error {
	if record.Type == "A" && (net.ParseIP(record.Value) == nil || strings.Contains(record.Value, ":")) {
		return errors.New("invalid IPv4")
	}
	if record.Type == "AAAA" && (net.ParseIP(record.Value) == nil || !strings.Contains(record.Value, ":")) {
		return errors.New("invalid IPv6")
	}
	return nil
}
