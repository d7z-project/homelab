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
	groupCache      *lru.Cache[string, *models.SiteGroup]
	exportCache     *lru.Cache[string, *models.SiteExport]
	groupListCache  *lru.Cache[string, []models.SiteGroup]
	exportListCache *lru.Cache[string, []models.SiteExport]
)

func init() {
	groupCache, _ = lru.New[string, *models.SiteGroup](512)
	exportCache, _ = lru.New[string, *models.SiteExport](512)
	groupListCache, _ = lru.New[string, []models.SiteGroup](16)
	exportListCache, _ = lru.New[string, []models.SiteExport](16)
}

func ClearCache() {
	groupCache.Purge()
	exportCache.Purge()
	groupListCache.Purge()
	exportListCache.Purge()
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
		groupListCache.Purge()
	}
	return err
}

func DeleteGroup(ctx context.Context, id string) error {
	_, err := common.DB.Child("network", "site", "groups").Delete(ctx, id)
	if err == nil {
		groupCache.Remove(id)
		groupListCache.Purge()
	}
	return err
}

func ListGroups(ctx context.Context, page, pageSize int, search string) ([]models.SiteGroup, int, error) {
	var all []models.SiteGroup
	if val, ok := groupListCache.Get("all"); ok {
		all = val
	} else {
		db := common.DB.Child("network", "site", "groups")
		items, err := db.List(ctx, "")
		if err != nil {
			return nil, 0, err
		}
		for _, v := range items {
			var group models.SiteGroup
			if err := json.Unmarshal([]byte(v.Value), &group); err == nil {
				all = append(all, group)
				groupCache.Add(group.ID, &group)
			}
		}
		groupListCache.Add("all", all)
	}

	res := make([]models.SiteGroup, 0)
	search = strings.ToLower(search)
	for _, g := range all {
		if search == "" || strings.Contains(strings.ToLower(g.Name), search) || strings.Contains(strings.ToLower(g.ID), search) {
			res = append(res, g)
		}
	}

	total := len(res)
	start := (page - 1) * pageSize
	if start < 0 { start = 0 }
	if start >= total { return []models.SiteGroup{}, total, nil }
	end := start + pageSize
	if end > total { end = total }
	return res[start:end], total, nil
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
		exportListCache.Purge()
	}
	return err
}

func DeleteExport(ctx context.Context, id string) error {
	_, err := common.DB.Child("network", "site", "exports").Delete(ctx, id)
	if err == nil {
		exportCache.Remove(id)
		exportListCache.Purge()
	}
	return err
}

func ListExports(ctx context.Context, page, pageSize int, search string) ([]models.SiteExport, int, error) {
	var all []models.SiteExport
	if val, ok := exportListCache.Get("all"); ok {
		all = val
	} else {
		db := common.DB.Child("network", "site", "exports")
		items, err := db.List(ctx, "")
		if err != nil {
			return nil, 0, err
		}
		for _, v := range items {
			var export models.SiteExport
			if err := json.Unmarshal([]byte(v.Value), &export); err == nil {
				all = append(all, export)
				exportCache.Add(export.ID, &export)
			}
		}
		exportListCache.Add("all", all)
	}

	res := make([]models.SiteExport, 0)
	search = strings.ToLower(search)
	for _, e := range all {
		if search == "" || strings.Contains(strings.ToLower(e.Name), search) || strings.Contains(strings.ToLower(e.ID), search) {
			res = append(res, e)
		}
	}

	total := len(res)
	start := (page - 1) * pageSize
	if start < 0 { start = 0 }
	if start >= total { return []models.SiteExport{}, total, nil }
	end := start + pageSize
	if end > total { end = total }
	return res[start:end], total, nil
}
