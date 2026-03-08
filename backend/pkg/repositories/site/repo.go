package site

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
	groupCache  *lru.Cache[string, *models.SiteGroup]
	exportCache *lru.Cache[string, *models.SiteExport]
)

func init() {
	groupCache, _ = lru.New[string, *models.SiteGroup](512)
	exportCache, _ = lru.New[string, *models.SiteExport](512)
}

func ClearCache() {
	groupCache.Purge()
	exportCache.Purge()
}

// Group Repo

func GetGroup(ctx context.Context, id string) (*models.SiteGroup, error) {
	if val, ok := groupCache.Get(id); ok {
		return val, nil
	}
	db := common.DB.Child("network", "site", "groups")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var group models.SiteGroup
	if err := json.Unmarshal([]byte(data), &group); err != nil {
		return nil, err
	}
	groupCache.Add(id, &group)
	return &group, nil
}

func SaveGroup(ctx context.Context, group *models.SiteGroup) error {
	db := common.DB.Child("network", "site", "groups")
	data, err := json.Marshal(group)
	if err != nil {
		return err
	}
	err = db.Put(ctx, group.ID, string(data), kv.TTLKeep)
	if err == nil {
		groupCache.Add(group.ID, group)
	}
	return err
}

func DeleteGroup(ctx context.Context, id string) error {
	_, err := common.DB.Child("network", "site", "groups").Delete(ctx, id)
	if err == nil {
		groupCache.Remove(id)
	}
	return err
}

func ScanGroups(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.SiteGroup], error) {
	db := common.DB.Child("network", "site", "groups")
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.SiteGroup, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var group models.SiteGroup
		if err := json.Unmarshal([]byte(v.Value), &group); err == nil {
			if search == "" || strings.Contains(strings.ToLower(group.Name), search) || strings.Contains(strings.ToLower(group.ID), search) {
				res = append(res, group)
				groupCache.Add(group.ID, &group)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.SiteGroup]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
			}, nil
		}
	}
	return &models.PaginationResponse[models.SiteGroup]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
	}, nil
}

// Export Repo

func GetExport(ctx context.Context, id string) (*models.SiteExport, error) {
	if val, ok := exportCache.Get(id); ok {
		return val, nil
	}
	db := common.DB.Child("network", "site", "exports")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var export models.SiteExport
	if err := json.Unmarshal([]byte(data), &export); err != nil {
		return nil, err
	}
	exportCache.Add(id, &export)
	return &export, nil
}

func SaveExport(ctx context.Context, export *models.SiteExport) error {
	db := common.DB.Child("network", "site", "exports")
	data, err := json.Marshal(export)
	if err != nil {
		return err
	}
	err = db.Put(ctx, export.ID, string(data), kv.TTLKeep)
	if err == nil {
		exportCache.Add(export.ID, export)
	}
	return err
}

func DeleteExport(ctx context.Context, id string) error {
	_, err := common.DB.Child("network", "site", "exports").Delete(ctx, id)
	if err == nil {
		exportCache.Remove(id)
	}
	return err
}

func ScanExports(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.SiteExport], error) {
	db := common.DB.Child("network", "site", "exports")
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.SiteExport, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var export models.SiteExport
		if err := json.Unmarshal([]byte(v.Value), &export); err == nil {
			if search == "" || strings.Contains(strings.ToLower(export.Name), search) || strings.Contains(strings.ToLower(export.ID), search) {
				res = append(res, export)
				exportCache.Add(export.ID, &export)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.SiteExport]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
			}, nil
		}
	}
	return &models.PaginationResponse[models.SiteExport]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
	}, nil
}
