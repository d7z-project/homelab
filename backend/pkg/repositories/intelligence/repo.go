package intelligence

import (
	"context"
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"

	"gopkg.d7z.net/middleware/kv"
)

func GetSource(ctx context.Context, id string) (*models.IntelligenceSource, error) {
	db := common.DB.Child("network", "intelligence", "sources")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var source models.IntelligenceSource
	if err := json.Unmarshal([]byte(data), &source); err != nil {
		return nil, err
	}
	return &source, nil
}

func SaveSource(ctx context.Context, source *models.IntelligenceSource) error {
	db := common.DB.Child("network", "intelligence", "sources")
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}
	return db.Put(ctx, source.ID, string(data), kv.TTLKeep)
}

func ScanAllSources(ctx context.Context) ([]models.IntelligenceSource, error) {
	db := common.DB.Child("network", "intelligence", "sources")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	res := make([]models.IntelligenceSource, 0, len(items))
	for _, v := range items {
		var s models.IntelligenceSource
		if err := json.Unmarshal([]byte(v.Value), &s); err == nil {
			res = append(res, s)
		}
	}
	return res, nil
}

func ScanSources(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.IntelligenceSource], error) {
	db := common.DB.Child("network", "intelligence", "sources")
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.IntelligenceSource, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var s models.IntelligenceSource
		if err := json.Unmarshal([]byte(v.Value), &s); err == nil {
			if search == "" || strings.Contains(strings.ToLower(s.Name), search) || strings.Contains(strings.ToLower(s.ID), search) {
				res = append(res, s)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.IntelligenceSource]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
			}, nil
		}
	}
	return &models.PaginationResponse[models.IntelligenceSource]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
	}, nil
}

func DeleteSource(ctx context.Context, id string) error {
	db := common.DB.Child("network", "intelligence", "sources")
	_, err := db.Delete(ctx, id)
	return err
}
