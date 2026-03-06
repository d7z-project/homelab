package intelligence

import (
	"context"
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"

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

func ListSources(ctx context.Context) ([]models.IntelligenceSource, error) {
	db := common.DB.Child("network", "intelligence", "sources")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	res := make([]models.IntelligenceSource, 0)
	for _, v := range items {
		var s models.IntelligenceSource
		if err := json.Unmarshal([]byte(v.Value), &s); err == nil {
			res = append(res, s)
		}
	}
	return res, nil
}

func DeleteSource(ctx context.Context, id string) error {
	db := common.DB.Child("network", "intelligence", "sources")
	_, err := db.Delete(ctx, id)
	return err
}
