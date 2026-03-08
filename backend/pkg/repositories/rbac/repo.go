package rbac

import (
	"context"
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
	"gopkg.d7z.net/middleware/kv"
)

var (
	roleCache *lru.Cache[string, *models.Role]

	rbacLastModified time.Time
	rbacMu           sync.RWMutex
)

func init() {
	roleCache, _ = lru.New[string, *models.Role](1024)
}

func GetLastModified() time.Time {
	val, err := common.DB.Child("auth", "rbac").Get(context.Background(), "last_modified")
	if err == nil && val != "" {
		t, err := time.Parse(time.RFC3339, val)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func updateLastModified() {
	now := time.Now().Format(time.RFC3339)
	_ = common.DB.Child("auth", "rbac").Put(context.Background(), "last_modified", now, kv.TTLKeep)
}

func ClearCache() {
	roleCache.Purge()
	updateLastModified()
}

func InvalidateCache(roleID string) {
	if roleID != "" {
		roleCache.Remove(roleID)
	}
	updateLastModified()
}

// ServiceAccount Repo

func GetServiceAccount(ctx context.Context, id string) (*models.ServiceAccount, error) {
	db := common.DB.Child("auth", "serviceaccounts")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var sa models.ServiceAccount
	if err := json.Unmarshal([]byte(data), &sa); err != nil {
		return nil, err
	}
	return &sa, nil
}

func SaveServiceAccount(ctx context.Context, sa *models.ServiceAccount) error {
	db := common.DB.Child("auth", "serviceaccounts")
	data, err := json.Marshal(sa)
	if err != nil {
		return err
	}
	return db.Put(ctx, sa.ID, string(data), kv.TTLKeep)
}

func DeleteServiceAccount(ctx context.Context, id string) error {
	_, err := common.DB.Child("auth", "serviceaccounts").Delete(ctx, id)
	return err
}

func ScanServiceAccounts(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.ServiceAccount], error) {
	db := common.DB.Child("auth", "serviceaccounts")
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.ServiceAccount, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var sa models.ServiceAccount
		if err := json.Unmarshal([]byte(v.Value), &sa); err == nil {
			if search == "" || strings.Contains(strings.ToLower(sa.Name), search) || strings.Contains(strings.ToLower(sa.ID), search) {
				res = append(res, sa)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.ServiceAccount]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
			}, nil
		}
	}
	return &models.PaginationResponse[models.ServiceAccount]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
	}, nil
}

// Role Repo

func GetRole(ctx context.Context, id string) (*models.Role, error) {
	// Check distributed last modified to invalidate local cache if needed
	remoteLM := GetLastModified()
	rbacMu.RLock()
	localLM := rbacLastModified
	rbacMu.RUnlock()

	if remoteLM.After(localLM) {
		ClearCache()
		rbacMu.Lock()
		rbacLastModified = remoteLM
		rbacMu.Unlock()
	}

	if val, ok := roleCache.Get(id); ok {
		return val, nil
	}
	db := common.DB.Child("auth", "roles")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var role models.Role
	if err := json.Unmarshal([]byte(data), &role); err != nil {
		return nil, err
	}
	roleCache.Add(id, &role)
	return &role, nil
}

func SaveRole(ctx context.Context, role *models.Role) error {
	db := common.DB.Child("auth", "roles")
	data, err := json.Marshal(role)
	if err != nil {
		return err
	}
	err = db.Put(ctx, role.ID, string(data), kv.TTLKeep)
	if err == nil {
		roleCache.Add(role.ID, role)
		InvalidateCache("")
	}
	return err
}

func DeleteRole(ctx context.Context, id string) error {
	_, err := common.DB.Child("auth", "roles").Delete(ctx, id)
	if err == nil {
		InvalidateCache(id)
	}
	return err
}

func ScanRoles(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.Role], error) {
	db := common.DB.Child("auth", "roles")
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.Role, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var role models.Role
		if err := json.Unmarshal([]byte(v.Value), &role); err == nil {
			if search == "" || strings.Contains(strings.ToLower(role.Name), search) || strings.Contains(strings.ToLower(role.ID), search) {
				res = append(res, role)
				roleCache.Add(role.ID, &role)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.Role]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
			}, nil
		}
	}
	return &models.PaginationResponse[models.Role]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
	}, nil
}

// RoleBinding Repo

func GetRoleBinding(ctx context.Context, id string) (*models.RoleBinding, error) {
	db := common.DB.Child("auth", "rolebindings")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var rb models.RoleBinding
	if err := json.Unmarshal([]byte(data), &rb); err != nil {
		return nil, err
	}
	return &rb, nil
}

func SaveRoleBinding(ctx context.Context, rb *models.RoleBinding) error {
	db := common.DB.Child("auth", "rolebindings")
	data, err := json.Marshal(rb)
	if err != nil {
		return err
	}
	err = db.Put(ctx, rb.ID, string(data), kv.TTLKeep)
	if err == nil {
		InvalidateCache("")
	}
	return err
}

func DeleteRoleBinding(ctx context.Context, id string) error {
	_, err := common.DB.Child("auth", "rolebindings").Delete(ctx, id)
	if err == nil {
		InvalidateCache("")
	}
	return err
}

func ScanRoleBindings(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.RoleBinding], error) {
	db := common.DB.Child("auth", "rolebindings")
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.RoleBinding, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var rb models.RoleBinding
		if err := json.Unmarshal([]byte(v.Value), &rb); err == nil {
			if search == "" || strings.Contains(strings.ToLower(rb.Name), search) || strings.Contains(strings.ToLower(rb.ID), search) {
				res = append(res, rb)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.RoleBinding]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
			}, nil
		}
	}
	return &models.PaginationResponse[models.RoleBinding]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
	}, nil
}

func ScanAllRoleBindings(ctx context.Context) ([]models.RoleBinding, error) {
	db := common.DB.Child("auth", "rolebindings")
	resp, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}

	res := make([]models.RoleBinding, 0)
	for _, v := range resp {
		var rb models.RoleBinding
		if err := json.Unmarshal([]byte(v.Value), &rb); err == nil {
			res = append(res, rb)
		}
	}
	return res, nil
}
