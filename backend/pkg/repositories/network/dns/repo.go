package dns

import (
	"context"
	"homelab/pkg/common"
	"strings"
	"time"

	dnsmodel "homelab/pkg/models/network/dns"
	"homelab/pkg/models/shared"

	"gopkg.d7z.net/middleware/kv"
)

var DomainRepo = common.NewBaseRepository[dnsmodel.DomainV1Meta, dnsmodel.DomainV1Status]("network", "Domain")
var RecordRepo = common.NewBaseRepository[dnsmodel.RecordV1Meta, dnsmodel.RecordV1Status]("network", "Record")

func GetLastModified() time.Time {
	val, err := common.DB.Child("network", "dns").Get(context.Background(), "last_modified")
	if err == nil && val != "" {
		t, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func ClearCache() {
	updateLastModified()
}

func updateLastModified() {
	now := time.Now().Format(time.RFC3339)
	_ = common.DB.Child("network", "dns").Put(context.Background(), "last_modified", now, kv.TTLKeep)
}

func GetDomain(ctx context.Context, id string) (*dnsmodel.Domain, error) {
	return DomainRepo.Get(ctx, id)
}

func SaveDomain(ctx context.Context, domain *dnsmodel.Domain) error {
	return DomainRepo.Cow(ctx, domain.ID, func(res *dnsmodel.Domain) error {
		res.Meta = domain.Meta
		res.Status = domain.Status
		return nil
	})
}

func DeleteDomain(ctx context.Context, id string) error {
	return DomainRepo.Delete(ctx, id)
}

func ScanDomains(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[dnsmodel.Domain], error) {
	search = strings.ToLower(search)
	return DomainRepo.List(ctx, cursor, limit, func(s *dnsmodel.Domain) bool {
		return search == "" || strings.Contains(strings.ToLower(s.Meta.Name), search) || strings.Contains(strings.ToLower(s.ID), search)
	})
}

func GetRecord(ctx context.Context, id string) (*dnsmodel.Record, error) {
	return RecordRepo.Get(ctx, id)
}

func SaveRecord(ctx context.Context, record *dnsmodel.Record) error {
	return RecordRepo.Cow(ctx, record.ID, func(res *dnsmodel.Record) error {
		res.Meta = record.Meta
		res.Status = record.Status
		return nil
	})
}

func DeleteRecord(ctx context.Context, id string) error {
	return RecordRepo.Delete(ctx, id)
}

func ScanRecords(ctx context.Context, domainID string, cursor string, limit int, search string) (*shared.PaginationResponse[dnsmodel.Record], error) {
	search = strings.ToLower(search)
	return RecordRepo.List(ctx, cursor, limit, func(s *dnsmodel.Record) bool {
		return (domainID == "" || s.Meta.DomainID == domainID) && (search == "" || strings.Contains(strings.ToLower(s.Meta.Name), search) || strings.Contains(strings.ToLower(s.Meta.Value), search) || strings.Contains(strings.ToLower(s.ID), search))
	})
}

func DeleteRecordsByDomain(ctx context.Context, domainID string) error {
	res, err := RecordRepo.List(ctx, "", 100000, func(s *dnsmodel.Record) bool {
		return s.Meta.DomainID == domainID
	})
	if err != nil {
		return err
	}
	for _, r := range res.Items {
		_ = RecordRepo.Delete(ctx, r.ID)
	}
	return nil
}
