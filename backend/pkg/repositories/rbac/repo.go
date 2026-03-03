package rbac

import (
	"context"
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
	"gopkg.d7z.net/middleware/kv"
)

var (
	roleCache *lru.Cache[string, *models.Role]

	rbCache struct {
		rbs   []models.RoleBinding
		valid bool
		mu    sync.RWMutex
	}
)

func init() {
	roleCache, _ = lru.New[string, *models.Role](1024)
}

func ClearCache() {
	roleCache.Purge()
	rbCache.mu.Lock()
	rbCache.valid = false
	rbCache.mu.Unlock()
}

func InvalidateCache(roleID string) {
	if roleID != "" {
		roleCache.Remove(roleID)
	}
	rbCache.mu.Lock()
	rbCache.valid = false
	rbCache.mu.Unlock()
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

func ListServiceAccounts(ctx context.Context, page uint64, pageSize uint, search string) ([]models.ServiceAccount, int64, error) {
	db := common.DB.Child("auth", "serviceaccounts")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, 0, err
	}
	var res []models.ServiceAccount
	search = strings.ToLower(search)
	for _, v := range items {
		var sa models.ServiceAccount
		if err := json.Unmarshal([]byte(v.Value), &sa); err == nil {
			if search == "" || strings.Contains(strings.ToLower(sa.Name), search) || strings.Contains(strings.ToLower(sa.ID), search) {
				res = append(res, sa)
			}
		}
	}
	total := int64(len(res))
	start := int(page) * int(pageSize)
	if start >= len(res) {
		return []models.ServiceAccount{}, total, nil
	}
	end := start + int(pageSize)
	if end > len(res) {
		end = len(res)
	}
	return res[start:end], total, nil
}

// Role Repo

func GetRole(ctx context.Context, id string) (*models.Role, error) {
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

func ListRoles(ctx context.Context, page uint64, pageSize uint, search string) ([]models.Role, int64, error) {
	db := common.DB.Child("auth", "roles")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, 0, err
	}
	var res []models.Role
	search = strings.ToLower(search)
	for _, v := range items {
		var role models.Role
		if err := json.Unmarshal([]byte(v.Value), &role); err == nil {
			if search == "" || strings.Contains(strings.ToLower(role.Name), search) || strings.Contains(strings.ToLower(role.ID), search) {
				res = append(res, role)
			}
		}
	}
	total := int64(len(res))
	start := int(page) * int(pageSize)
	if start >= len(res) {
		return []models.Role{}, total, nil
	}
	end := start + int(pageSize)
	if end > len(res) {
		end = len(res)
	}
	return res[start:end], total, nil
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

func ListRoleBindings(ctx context.Context, page uint64, pageSize uint, search string) ([]models.RoleBinding, int64, error) {
	db := common.DB.Child("auth", "rolebindings")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, 0, err
	}
	var res []models.RoleBinding
	search = strings.ToLower(search)
	for _, v := range items {
		var rb models.RoleBinding
		if err := json.Unmarshal([]byte(v.Value), &rb); err == nil {
			if search == "" || strings.Contains(strings.ToLower(rb.Name), search) || strings.Contains(strings.ToLower(rb.ID), search) {
				res = append(res, rb)
			}
		}
	}
	total := int64(len(res))
	start := int(page) * int(pageSize)
	if start >= len(res) {
		return []models.RoleBinding{}, total, nil
	}
	end := start + int(pageSize)
	if end > len(res) {
		end = len(res)
	}
	return res[start:end], total, nil
}

func ListRoleBindingsAll(ctx context.Context) ([]models.RoleBinding, error) {
	rbCache.mu.RLock()
	if rbCache.valid {
		res := rbCache.rbs
		rbCache.mu.RUnlock()
		return res, nil
	}
	rbCache.mu.RUnlock()

	db := common.DB.Child("auth", "rolebindings")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	var res []models.RoleBinding
	for _, v := range items {
		var rb models.RoleBinding
		if err := json.Unmarshal([]byte(v.Value), &rb); err == nil {
			res = append(res, rb)
		}
	}
	rbCache.mu.Lock()
	rbCache.rbs = res
	rbCache.valid = true
	rbCache.mu.Unlock()
	return res, nil
}
