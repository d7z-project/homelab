package dns

import (
	"context"
	"homelab/pkg/common"
	runtimepkg "homelab/pkg/runtime"
	"strings"
	"time"

	dnsmodel "homelab/pkg/models/network/dns"
	"homelab/pkg/models/shared"

	"gopkg.d7z.net/middleware/kv"
)

var domainRepo = common.NewBaseRepository[dnsmodel.DomainV1Meta, dnsmodel.DomainV1Status]("network", "Domain")
var recordRepo = common.NewBaseRepository[dnsmodel.RecordV1Meta, dnsmodel.RecordV1Status]("network", "Record")

func GetLastModified(ctx context.Context) time.Time {
	db := runtimepkg.DBFromContext(ctx)
	if db == nil {
		return time.Time{}
	}
	val, err := db.Child("network", "dns").Get(ctx, "last_modified")
	if err == nil && val != "" {
		t, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func ClearCache(ctx context.Context) {
	updateLastModified(ctx)
}

func updateLastModified(ctx context.Context) {
	db := runtimepkg.DBFromContext(ctx)
	if db == nil {
		return
	}
	now := time.Now().Format(time.RFC3339)
	_ = db.Child("network", "dns").Put(ctx, "last_modified", now, kv.TTLKeep)
}

func GetDomain(ctx context.Context, id string) (*dnsmodel.Domain, error) {
	return domainRepo.Get(ctx, id)
}

func SaveDomain(ctx context.Context, domain *dnsmodel.Domain) error {
	return domainRepo.Save(ctx, domain)
}

func DeleteDomain(ctx context.Context, id string) error {
	return domainRepo.Delete(ctx, id)
}

func ScanDomains(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[dnsmodel.Domain], error) {
	search = strings.ToLower(search)
	return domainRepo.List(ctx, cursor, limit, func(s *dnsmodel.Domain) bool {
		return search == "" || strings.Contains(strings.ToLower(s.Meta.Name), search) || strings.Contains(strings.ToLower(s.ID), search)
	})
}

func GetRecord(ctx context.Context, id string) (*dnsmodel.Record, error) {
	return recordRepo.Get(ctx, id)
}

func SaveRecord(ctx context.Context, record *dnsmodel.Record) error {
	return recordRepo.Save(ctx, record)
}

func DeleteRecord(ctx context.Context, id string) error {
	return recordRepo.Delete(ctx, id)
}

func ScanRecords(ctx context.Context, domainID string, cursor string, limit int, search string) (*shared.PaginationResponse[dnsmodel.Record], error) {
	search = strings.ToLower(search)
	return recordRepo.List(ctx, cursor, limit, func(s *dnsmodel.Record) bool {
		return (domainID == "" || s.Meta.DomainID == domainID) && (search == "" || strings.Contains(strings.ToLower(s.Meta.Name), search) || strings.Contains(strings.ToLower(s.Meta.Value), search) || strings.Contains(strings.ToLower(s.ID), search))
	})
}

func DeleteRecordsByDomain(ctx context.Context, domainID string) error {
	res, err := ScanRecords(ctx, "", "", 100000, "")
	if err != nil {
		return err
	}
	for _, r := range res.Items {
		if r.Meta.DomainID != domainID {
			continue
		}
		_ = DeleteRecord(ctx, r.ID)
	}
	return nil
}

func ScanAllDomains(ctx context.Context) ([]dnsmodel.Domain, error) {
	return domainRepo.ListAll(ctx)
}

func ScanAllRecords(ctx context.Context) ([]dnsmodel.Record, error) {
	return recordRepo.ListAll(ctx)
}
