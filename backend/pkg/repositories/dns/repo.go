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
	domainCache     *lru.Cache[string, *models.Domain]
	recordCache     *lru.Cache[string, *models.Record]
	domainListCache *lru.Cache[string, []models.Domain]
	recordListCache *lru.Cache[string, []models.Record]
)

func init() {
	domainCache, _ = lru.New[string, *models.Domain](1024)
	recordCache, _ = lru.New[string, *models.Record](2048)
	domainListCache, _ = lru.New[string, []models.Domain](16)
	recordListCache, _ = lru.New[string, []models.Record](128)
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
	domainListCache.Purge()
	recordListCache.Purge()
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
		domainListCache.Purge() // Invalidate all lists
		updateLastModified()
	}
	return err
}

func DeleteDomain(ctx context.Context, id string) error {
	_, err := common.DB.Child("network/dns", "domains").Delete(ctx, id)
	if err == nil {
		domainCache.Remove(id)
		domainListCache.Purge() // Invalidate all lists
		updateLastModified()
	}
	return err
}

func ListDomains(ctx context.Context, page int, pageSize int, search string) ([]models.Domain, int, error) {
	var all []models.Domain
	if val, ok := domainListCache.Get("all"); ok {
		all = val
	} else {
		db := common.DB.Child("network/dns", "domains")
		items, err := db.List(ctx, "")
		if err != nil {
			return nil, 0, err
		}
		for _, v := range items {
			var domain models.Domain
			if err := json.Unmarshal([]byte(v.Value), &domain); err == nil {
				all = append(all, domain)
				domainCache.Add(domain.ID, &domain)
			}
		}
		domainListCache.Add("all", all)
	}

	res := make([]models.Domain, 0)
	search = strings.ToLower(search)
	for _, domain := range all {
		if search == "" || strings.Contains(strings.ToLower(domain.Name), search) || strings.Contains(strings.ToLower(domain.ID), search) {
			res = append(res, domain)
		}
	}

	total := len(res)
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []models.Domain{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return res[start:end], total, nil
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
		recordListCache.Purge() // Invalidate all record lists
		updateLastModified()
	}
	return err
}

func DeleteRecord(ctx context.Context, id string) error {
	_, err := common.DB.Child("network/dns", "records").Delete(ctx, id)
	if err == nil {
		recordCache.Remove(id)
		recordListCache.Purge() // Invalidate all record lists
		updateLastModified()
	}
	return err
}

func ListRecords(ctx context.Context, domainID string, page int, pageSize int, search string) ([]models.Record, int, error) {
	var all []models.Record
	cacheKey := "all"
	if domainID != "" {
		cacheKey = "domain:" + domainID
	}

	if val, ok := recordListCache.Get(cacheKey); ok {
		all = val
	} else {
		db := common.DB.Child("network/dns", "records")
		items, err := db.List(ctx, "")
		if err != nil {
			return nil, 0, err
		}
		for _, v := range items {
			var record models.Record
			if err := json.Unmarshal([]byte(v.Value), &record); err == nil {
				recordCache.Add(record.ID, &record)
				if domainID == "" || record.DomainID == domainID {
					all = append(all, record)
				}
			}
		}
		recordListCache.Add(cacheKey, all)
	}

	res := make([]models.Record, 0)
	search = strings.ToLower(search)
	for _, record := range all {
		if search == "" || strings.Contains(strings.ToLower(record.Name), search) || strings.Contains(strings.ToLower(record.Value), search) || strings.Contains(strings.ToLower(record.ID), search) {
			res = append(res, record)
		}
	}

	total := len(res)
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []models.Record{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return res[start:end], total, nil
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
		recordListCache.Purge()
		updateLastModified()
	}
	return nil
}
