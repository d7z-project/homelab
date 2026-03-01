package dns

import (
	"context"
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"

	lru "github.com/hashicorp/golang-lru/v2"
	"gopkg.d7z.net/middleware/kv"
)

var (
	domainCache *lru.Cache[string, *models.Domain]
)

func init() {
	domainCache, _ = lru.New[string, *models.Domain](1024)
}

// Domain Repo

func GetDomain(ctx context.Context, id string) (*models.Domain, error) {
	if val, ok := domainCache.Get(id); ok {
		return val, nil
	}
	db := common.DB.Child("dns", "domains")
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
	db := common.DB.Child("dns", "domains")
	data, err := json.Marshal(domain)
	if err != nil {
		return err
	}
	err = db.Put(ctx, domain.ID, string(data), kv.TTLKeep)
	if err == nil {
		domainCache.Add(domain.ID, domain)
	}
	return err
}

func DeleteDomain(ctx context.Context, id string) error {
	_, err := common.DB.Child("dns", "domains").Delete(ctx, id)
	if err == nil {
		domainCache.Remove(id)
	}
	return err
}

func ListDomains(ctx context.Context, page int, pageSize int, search string) ([]models.Domain, int, error) {
	db := common.DB.Child("dns", "domains")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, 0, err
	}
	var res []models.Domain
	search = strings.ToLower(search)
	for _, v := range items {
		var domain models.Domain
		if err := json.Unmarshal([]byte(v.Value), &domain); err == nil {
			if search == "" || strings.Contains(strings.ToLower(domain.Name), search) {
				res = append(res, domain)
			}
		}
	}
	total := len(res)
	start := page * pageSize
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
	db := common.DB.Child("dns", "records")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var record models.Record
	if err := json.Unmarshal([]byte(data), &record); err != nil {
		return nil, err
	}
	return &record, nil
}

func SaveRecord(ctx context.Context, record *models.Record) error {
	db := common.DB.Child("dns", "records")
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	return db.Put(ctx, record.ID, string(data), kv.TTLKeep)
}

func DeleteRecord(ctx context.Context, id string) error {
	_, err := common.DB.Child("dns", "records").Delete(ctx, id)
	return err
}

func ListRecords(ctx context.Context, domainID string, page int, pageSize int, search string) ([]models.Record, int, error) {
	db := common.DB.Child("dns", "records")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, 0, err
	}
	var res []models.Record
	search = strings.ToLower(search)
	for _, v := range items {
		var record models.Record
		if err := json.Unmarshal([]byte(v.Value), &record); err == nil {
			// Filter by domainID if provided
			if domainID != "" && record.DomainID != domainID {
				continue
			}
			if search == "" || strings.Contains(strings.ToLower(record.Name), search) || strings.Contains(strings.ToLower(record.Value), search) {
				res = append(res, record)
			}
		}
	}
	total := len(res)
	start := page * pageSize
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
	db := common.DB.Child("dns", "records")
	items, err := db.List(ctx, "")
	if err != nil {
		return err
	}
	for _, v := range items {
		var record models.Record
		if err := json.Unmarshal([]byte(v.Value), &record); err == nil {
			if record.DomainID == domainID {
				db.Delete(ctx, record.ID)
			}
		}
	}
	return nil
}
