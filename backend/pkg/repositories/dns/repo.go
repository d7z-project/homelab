package dns

import (
	"context"
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"gopkg.d7z.net/middleware/kv"
)

var (
	domainCache *lru.Cache[string, *models.Domain]
	recordCache *lru.Cache[string, *models.Record]
)

func init() {
	domainCache, _ = lru.New[string, *models.Domain](1024)
	recordCache, _ = lru.New[string, *models.Record](2048)
}

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
	domainCache.Purge()
	recordCache.Purge()
	updateLastModified()
}

func updateLastModified() {
	now := time.Now().Format(time.RFC3339)
	_ = common.DB.Child("network", "dns").Put(context.Background(), "last_modified", now, kv.TTLKeep)
}

// Domain Repo

func GetDomain(ctx context.Context, id string) (*models.Domain, error) {
	if val, ok := domainCache.Get(id); ok {
		return val, nil
	}
	db := common.DB.Child("network/dns", "domains")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var domain models.Domain
	if err := json.Unmarshal([]byte(data), &domain); err != nil {
		return nil, err
	}
	domainCache.Add(id, &domain)
	return &domain, nil
}

func SaveDomain(ctx context.Context, domain *models.Domain) error {
	db := common.DB.Child("network/dns", "domains")
	data, err := json.Marshal(domain)
	if err != nil {
		return err
	}
	err = db.Put(ctx, domain.ID, string(data), kv.TTLKeep)
	if err == nil {
		domainCache.Add(domain.ID, domain)
		updateLastModified()
	}
	return err
}

func DeleteDomain(ctx context.Context, id string) error {
	_, err := common.DB.Child("network/dns", "domains").Delete(ctx, id)
	if err == nil {
		domainCache.Remove(id)
		updateLastModified()
	}
	return err
}

func ScanDomains(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.Domain], error) {
	db := common.DB.Child("network/dns", "domains")
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.Domain, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var domain models.Domain
		if err := json.Unmarshal([]byte(v.Value), &domain); err == nil {
			if search == "" || strings.Contains(strings.ToLower(domain.Name), search) || strings.Contains(strings.ToLower(domain.ID), search) {
				res = append(res, domain)
				domainCache.Add(domain.ID, &domain)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.Domain]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
			}, nil
		}
	}

	return &models.PaginationResponse[models.Domain]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
	}, nil
}

// Record Repo

func GetRecord(ctx context.Context, id string) (*models.Record, error) {
	if val, ok := recordCache.Get(id); ok {
		return val, nil
	}
	db := common.DB.Child("network/dns", "records")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var record models.Record
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		return nil, err
	}
	recordCache.Add(id, &record)
	return &record, nil
}

func SaveRecord(ctx context.Context, record *models.Record) error {
	db := common.DB.Child("network/dns", "records")
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	err = db.Put(ctx, record.ID, string(data), kv.TTLKeep)
	if err == nil {
		recordCache.Add(record.ID, record)
		updateLastModified()
	}
	return err
}

func DeleteRecord(ctx context.Context, id string) error {
	_, err := common.DB.Child("network/dns", "records").Delete(ctx, id)
	if err == nil {
		recordCache.Remove(id)
		updateLastModified()
	}
	return err
}

func ScanRecords(ctx context.Context, domainID string, cursor string, limit int, search string) (*models.PaginationResponse[models.Record], error) {
	db := common.DB.Child("network/dns", "records")
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.Record, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var record models.Record
		if err := json.Unmarshal([]byte(v.Value), &record); err == nil {
			if (domainID == "" || record.DomainID == domainID) && (search == "" || strings.Contains(strings.ToLower(record.Name), search) || strings.Contains(strings.ToLower(record.Value), search) || strings.Contains(strings.ToLower(record.ID), search)) {
				res = append(res, record)
				recordCache.Add(record.ID, &record)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.Record]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
			}, nil
		}
	}

	return &models.PaginationResponse[models.Record]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
	}, nil
}

func DeleteRecordsByDomain(ctx context.Context, domainID string) error {
	db := common.DB.Child("network/dns", "records")
	items, err := db.List(ctx, "")
	if err != nil {
		return err
	}
	deleted := false
	for _, v := range items {
		var record models.Record
		if err := json.Unmarshal([]byte(v.Value), &record); err == nil {
			if record.DomainID == domainID {
				db.Delete(ctx, record.ID)
				recordCache.Remove(record.ID)
				deleted = true
			}
		}
	}
	if deleted {
		updateLastModified()
	}
	return nil
}
