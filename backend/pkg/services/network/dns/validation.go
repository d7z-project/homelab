package dns

import (
	"context"
	"errors"
	"strings"

	dnsmodel "homelab/pkg/models/network/dns"
)

func normalizeDomain(domain *dnsmodel.Domain) error {
	if domain == nil {
		return errors.New("domain is required")
	}
	domain.Meta.Name = strings.TrimSpace(domain.Meta.Name)
	if domain.Meta.Name == "" {
		return errors.New("domain name is required")
	}
	return domain.Meta.Validate(context.Background())
}

func normalizeRecord(record *dnsmodel.Record) error {
	if record == nil {
		return errors.New("record is required")
	}
	record.Meta.Name = strings.TrimSpace(record.Meta.Name)
	record.Meta.Type = strings.TrimSpace(record.Meta.Type)
	record.Meta.Value = strings.TrimSpace(record.Meta.Value)
	record.Meta.Comments = strings.TrimSpace(record.Meta.Comments)
	if record.Meta.Name == "" {
		return errors.New("record name is required")
	}
	if record.Meta.Type == "" {
		return errors.New("record type is required")
	}
	if record.Meta.Value == "" {
		return errors.New("record value is required")
	}
	return record.Meta.Validate(context.Background())
}
