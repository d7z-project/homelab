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
	groupCache  *lru.Cache[string, *models.IPGroup]
	exportCache *lru.Cache[string, *models.IPExport]
	policyCache *lru.Cache[string, *models.IPSyncPolicy]
)

func init() {
	groupCache, _ = lru.New[string, *models.IPGroup](512)
	exportCache, _ = lru.New[string, *models.IPExport](512)
	policyCache, _ = lru.New[string, *models.IPSyncPolicy](512)
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
		updateLastModified()
	}
	return err
}

func DeleteGroup(ctx context.Context, id string) error {
	_, err := common.DB.Child("network", "ip", "groups").Delete(ctx, id)
	if err == nil {
		groupCache.Remove(id)
		updateLastModified()
	}
	return err
}

func ScanGroups(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.IPGroup], error) {
	db := common.DB.Child("network", "ip", "groups")
	count, _ := db.Count(ctx)
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.IPGroup, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var group models.IPGroup
		if err := json.Unmarshal([]byte(v.Value), &group); err == nil {
			if search == "" || strings.Contains(strings.ToLower(group.Name), search) || strings.Contains(strings.ToLower(group.ID), search) {
				res = append(res, group)
				groupCache.Add(group.ID, &group)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.IPGroup]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
				Total:      int64(count),
			}, nil
		}
	}

	return &models.PaginationResponse[models.IPGroup]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
		Total:      int64(count),
	}, nil
}

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
		updateLastModified()
	}
	return err
}

func DeleteExport(ctx context.Context, id string) error {
	_, err := common.DB.Child("network", "ip", "exports").Delete(ctx, id)
	if err == nil {
		exportCache.Remove(id)
		updateLastModified()
	}
	return err
}

func ScanExports(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.IPExport], error) {
	db := common.DB.Child("network", "ip", "exports")
	count, _ := db.Count(ctx)
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.IPExport, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var export models.IPExport
		if err := json.Unmarshal([]byte(v.Value), &export); err == nil {
			if search == "" || strings.Contains(strings.ToLower(export.Name), search) || strings.Contains(strings.ToLower(export.ID), search) {
				res = append(res, export)
				exportCache.Add(export.ID, &export)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.IPExport]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
				Total:      int64(count),
			}, nil
		}
	}

	return &models.PaginationResponse[models.IPExport]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
		Total:      int64(count),
	}, nil
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
		updateLastModified()
	}
	return err
}

func DeleteSyncPolicy(ctx context.Context, id string) error {
	_, err := common.DB.Child("network", "ip", "policies").Delete(ctx, id)
	if err == nil {
		policyCache.Remove(id)
		updateLastModified()
	}
	return err
}

func ScanSyncPolicies(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.IPSyncPolicy], error) {
	db := common.DB.Child("network", "ip", "policies")
	count, _ := db.Count(ctx)
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.IPSyncPolicy, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var policy models.IPSyncPolicy
		if err := json.Unmarshal([]byte(v.Value), &policy); err == nil {
			if search == "" || strings.Contains(strings.ToLower(policy.Name), search) || strings.Contains(strings.ToLower(policy.ID), search) {
				res = append(res, policy)
				policyCache.Add(policy.ID, &policy)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.IPSyncPolicy]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
				Total:      int64(count),
			}, nil
		}
	}

	return &models.PaginationResponse[models.IPSyncPolicy]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
		Total:      int64(count),
	}, nil
}
