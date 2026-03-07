package ip

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
	groupCache      *lru.Cache[string, *models.IPGroup]
	exportCache     *lru.Cache[string, *models.IPExport]
	policyCache     *lru.Cache[string, *models.IPSyncPolicy]
	groupListCache  *lru.Cache[string, []models.IPGroup]
	exportListCache *lru.Cache[string, []models.IPExport]
	policyListCache *lru.Cache[string, []models.IPSyncPolicy]
)

func init() {
	groupCache, _ = lru.New[string, *models.IPGroup](512)
	exportCache, _ = lru.New[string, *models.IPExport](512)
	policyCache, _ = lru.New[string, *models.IPSyncPolicy](512)
	groupListCache, _ = lru.New[string, []models.IPGroup](16)
	exportListCache, _ = lru.New[string, []models.IPExport](16)
	policyListCache, _ = lru.New[string, []models.IPSyncPolicy](16)
}

func updateLastModified() {
	now := time.Now().Format(time.RFC3339)
	_ = common.DB.Child("network", "ip").Put(context.Background(), "last_modified", now, kv.TTLKeep)
}

func GetLastModified() time.Time {
	val, err := common.DB.Child("network", "ip").Get(context.Background(), "last_modified")
	if err == nil && val != "" {
		t, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

// Group Repo

func GetGroup(ctx context.Context, id string) (*models.IPGroup, error) {
	if val, ok := groupCache.Get(id); ok {
		return val, nil
	}
	db := common.DB.Child("network", "ip", "groups")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var group models.IPGroup
	if err := json.Unmarshal([]byte(data), &group); err != nil {
		return nil, err
	}
	groupCache.Add(id, &group)
	return &group, nil
}

func SaveGroup(ctx context.Context, group *models.IPGroup) error {
	db := common.DB.Child("network", "ip", "groups")
	data, err := json.Marshal(group)
	if err != nil {
		return err
	}
	err = db.Put(ctx, group.ID, string(data), kv.TTLKeep)
	if err == nil {
		groupCache.Add(group.ID, group)
		groupListCache.Purge()
		updateLastModified()
	}
	return err
}

func DeleteGroup(ctx context.Context, id string) error {
	_, err := common.DB.Child("network", "ip", "groups").Delete(ctx, id)
	if err == nil {
		groupCache.Remove(id)
		groupListCache.Purge()
		updateLastModified()
	}
	return err
}

func ListGroups(ctx context.Context, page int, pageSize int, search string) ([]models.IPGroup, int, error) {
	var all []models.IPGroup
	if val, ok := groupListCache.Get("all"); ok {
		all = val
	} else {
		db := common.DB.Child("network", "ip", "groups")
		items, err := db.List(ctx, "")
		if err != nil {
			return nil, 0, err
		}
		for _, v := range items {
			var group models.IPGroup
			if err := json.Unmarshal([]byte(v.Value), &group); err == nil {
				all = append(all, group)
				groupCache.Add(group.ID, &group)
			}
		}
		groupListCache.Add("all", all)
	}

	res := make([]models.IPGroup, 0)
	search = strings.ToLower(search)
	for _, g := range all {
		if search == "" || strings.Contains(strings.ToLower(g.Name), search) || strings.Contains(strings.ToLower(g.ID), search) {
			res = append(res, g)
		}
	}

	total := len(res)
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []models.IPGroup{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return res[start:end], total, nil
}

// Export Repo

func GetExport(ctx context.Context, id string) (*models.IPExport, error) {
	if val, ok := exportCache.Get(id); ok {
		return val, nil
	}
	db := common.DB.Child("network", "ip", "exports")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var export models.IPExport
	if err := json.Unmarshal([]byte(data), &export); err != nil {
		return nil, err
	}
	exportCache.Add(id, &export)
	return &export, nil
}

func SaveExport(ctx context.Context, export *models.IPExport) error {
	db := common.DB.Child("network", "ip", "exports")
	data, err := json.Marshal(export)
	if err != nil {
		return err
	}
	err = db.Put(ctx, export.ID, string(data), kv.TTLKeep)
	if err == nil {
		exportCache.Add(export.ID, export)
		exportListCache.Purge()
		updateLastModified()
	}
	return err
}

func DeleteExport(ctx context.Context, id string) error {
	_, err := common.DB.Child("network", "ip", "exports").Delete(ctx, id)
	if err == nil {
		exportCache.Remove(id)
		exportListCache.Purge()
		updateLastModified()
	}
	return err
}

func ListExports(ctx context.Context, page int, pageSize int, search string) ([]models.IPExport, int, error) {
	var all []models.IPExport
	if val, ok := exportListCache.Get("all"); ok {
		all = val
	} else {
		db := common.DB.Child("network", "ip", "exports")
		items, err := db.List(ctx, "")
		if err != nil {
			return nil, 0, err
		}
		for _, v := range items {
			var export models.IPExport
			if err := json.Unmarshal([]byte(v.Value), &export); err == nil {
				all = append(all, export)
				exportCache.Add(export.ID, &export)
			}
		}
		exportListCache.Add("all", all)
	}

	res := make([]models.IPExport, 0)
	search = strings.ToLower(search)
	for _, e := range all {
		if search == "" || strings.Contains(strings.ToLower(e.Name), search) || strings.Contains(strings.ToLower(e.ID), search) {
			res = append(res, e)
		}
	}

	total := len(res)
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []models.IPExport{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return res[start:end], total, nil
}

// SyncPolicy Repo

func GetSyncPolicy(ctx context.Context, id string) (*models.IPSyncPolicy, error) {
	if val, ok := policyCache.Get(id); ok {
		return val, nil
	}
	db := common.DB.Child("network", "ip", "policies")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var policy models.IPSyncPolicy
	if err := json.Unmarshal([]byte(data), &policy); err != nil {
		return nil, err
	}
	policyCache.Add(id, &policy)
	return &policy, nil
}

func SaveSyncPolicy(ctx context.Context, policy *models.IPSyncPolicy) error {
	db := common.DB.Child("network", "ip", "policies")
	data, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	err = db.Put(ctx, policy.ID, string(data), kv.TTLKeep)
	if err == nil {
		policyCache.Add(policy.ID, policy)
		policyListCache.Purge()
		updateLastModified()
	}
	return err
}

func DeleteSyncPolicy(ctx context.Context, id string) error {
	_, err := common.DB.Child("network", "ip", "policies").Delete(ctx, id)
	if err == nil {
		policyCache.Remove(id)
		policyListCache.Purge()
		updateLastModified()
	}
	return err
}

func ListSyncPolicies(ctx context.Context, page int, pageSize int, search string) ([]models.IPSyncPolicy, int, error) {
	var all []models.IPSyncPolicy
	if val, ok := policyListCache.Get("all"); ok {
		all = val
	} else {
		db := common.DB.Child("network", "ip", "policies")
		items, err := db.List(ctx, "")
		if err != nil {
			return nil, 0, err
		}
		for _, v := range items {
			var policy models.IPSyncPolicy
			if err := json.Unmarshal([]byte(v.Value), &policy); err == nil {
				all = append(all, policy)
				policyCache.Add(policy.ID, &policy)
			}
		}
		policyListCache.Add("all", all)
	}

	res := make([]models.IPSyncPolicy, 0)
	search = strings.ToLower(search)
	for _, p := range all {
		if search == "" || strings.Contains(strings.ToLower(p.Name), search) || strings.Contains(strings.ToLower(p.ID), search) {
			res = append(res, p)
		}
	}

	total := len(res)
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []models.IPSyncPolicy{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return res[start:end], total, nil
}
