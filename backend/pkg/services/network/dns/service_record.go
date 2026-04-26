package dns

import (
	"context"
	"errors"
	"fmt"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	dnsmodel "homelab/pkg/models/network/dns"
	"homelab/pkg/models/shared"
	dnsrepo "homelab/pkg/repositories/network/dns"
	"net"
	"strings"

	"github.com/google/uuid"
)

func ScanRecords(ctx context.Context, domainID string, cursor string, limit int, search string) (*shared.PaginationResponse[dnsmodel.Record], error) {
	if domainID == "" {
		perms := commonauth.PermissionsFromContext(ctx)
		if perms.IsAllowed("network/dns") {
			return dnsrepo.ScanRecords(ctx, "", cursor, limit, search)
		}
		return &shared.PaginationResponse[dnsmodel.Record]{
			Items: []dnsmodel.Record{},
		}, nil
	}

	dom, err := dnsrepo.GetDomain(ctx, domainID)
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	if !perms.IsAllowed("network/dns") && !perms.IsAllowed("network/dns/"+dom.Meta.Name) {
		return nil, fmt.Errorf("%w: network/dns/%s", commonauth.ErrPermissionDenied, dom.Meta.Name)
	}
	return dnsrepo.ScanRecords(ctx, domainID, cursor, limit, search)
}

func GetRecord(ctx context.Context, id string) (*dnsmodel.Record, error) {
	return dnsrepo.GetRecord(ctx, id)
}

func CreateRecord(ctx context.Context, record *dnsmodel.Record) (*dnsmodel.Record, error) {
	if err := normalizeRecord(record); err != nil {
		return nil, err
	}
	dom, err := dnsrepo.GetDomain(ctx, record.Meta.DomainID)
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
	if err := dnsrepo.SaveRecord(ctx, record); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRecord", record.Meta.Name+"."+dom.Meta.Name, "Failed", false)
		return nil, err
	}
	updateSOASerial(ctx, dom.ID)
	commonaudit.FromContext(ctx).Log("CreateRecord", record.Meta.Name+"."+dom.Meta.Name, "Created", true)
	return record, nil
}

func UpdateRecord(ctx context.Context, id string, record *dnsmodel.Record) (*dnsmodel.Record, error) {
	if err := normalizeRecord(record); err != nil {
		return nil, err
	}
	existing, err := dnsrepo.GetRecord(ctx, id)
	if err != nil {
		return nil, errors.New("not found")
	}
	dom, _ := dnsrepo.GetDomain(ctx, existing.Meta.DomainID)
	if dom == nil {
		return nil, errors.New("domain not found")
	}

	if existing.Meta.Type == "SOA" {
		return nil, errors.New("SOA managed by system")
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

	existing.Meta.Name = record.Meta.Name
	existing.Meta.Type = record.Meta.Type
	existing.Meta.Value = record.Meta.Value
	existing.Meta.TTL = record.Meta.TTL
	existing.Meta.Enabled = record.Meta.Enabled
	existing.Meta.Comments = record.Meta.Comments
	err = dnsrepo.SaveRecord(ctx, existing)

	if err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRecord", record.Meta.Name+"."+dom.Meta.Name, "Failed", false)
		return nil, err
	}
	updateSOASerial(ctx, dom.ID)

	updated, _ := dnsrepo.GetRecord(ctx, id)
	commonaudit.FromContext(ctx).Log("UpdateRecord", record.Meta.Name+"."+dom.Meta.Name, "Updated", true)
	return updated, nil
}

func DeleteRecord(ctx context.Context, id string) error {
	existing, err := dnsrepo.GetRecord(ctx, id)
	if err != nil {
		return errors.New("not found")
	}
	dom, _ := dnsrepo.GetDomain(ctx, existing.Meta.DomainID)
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
	err = dnsrepo.DeleteRecord(ctx, id)
	if err == nil {
		updateSOASerial(ctx, dom.ID)
		commonaudit.FromContext(ctx).Log("DeleteRecord", existing.Meta.Name+"."+dom.Meta.Name, "Deleted", true)
	}
	return err
}

func validateRecord(ctx context.Context, record *dnsmodel.Record) error {
	if record.Meta.Type == "A" && (net.ParseIP(record.Meta.Value) == nil || strings.Contains(record.Meta.Value, ":")) {
		return errors.New("invalid IPv4")
	}
	if record.Meta.Type == "AAAA" && (net.ParseIP(record.Meta.Value) == nil || !strings.Contains(record.Meta.Value, ":")) {
		return errors.New("invalid IPv6")
	}
	return nil
}
