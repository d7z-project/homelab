package auth

import (
	"context"
	"encoding/json"
	"homelab/pkg/common"
	"sort"
	"strings"

	"gopkg.d7z.net/middleware/kv"
)

type PolicyRule struct {
	Verbs     []string `json:"verbs"`
	Resources []string `json:"resources"`
}

type Role struct {
	Name  string       `json:"name"`
	Rules []PolicyRule `json:"rules"`
}

type ServiceAccount struct {
	Name     string `json:"name"`
	Token    string `json:"token"`
	Comments string `json:"comments"`
}

type RoleBinding struct {
	Name               string   `json:"name"`
	RoleNames          []string `json:"roleNames"`
	ServiceAccountName string   `json:"serviceAccountName"`
	Enabled            bool     `json:"enabled"`
}

func GetServiceAccount(ctx context.Context, name string) (*ServiceAccount, error) {
	db := common.DB.Child("auth", "serviceaccounts")
	data, err := db.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	var sa ServiceAccount
	if err := json.Unmarshal([]byte(data), &sa); err != nil {
		return nil, err
	}
	return &sa, nil
}

func SaveServiceAccount(ctx context.Context, sa *ServiceAccount) error {
	db := common.DB.Child("auth", "serviceaccounts")
	data, err := json.Marshal(sa)
	if err != nil {
		return err
	}
	// Map token to name
	if sa.Token != "" {
		tokenDB := common.DB.Child("auth", "tokens")
		err = tokenDB.Put(ctx, sa.Token, sa.Name, kv.TTLKeep)
		if err != nil {
			return err
		}
	}
	return db.Put(ctx, sa.Name, string(data), kv.TTLKeep)
}

func DeleteServiceAccount(ctx context.Context, name string) error {
	sa, err := GetServiceAccount(ctx, name)
	if err == nil && sa.Token != "" {
		common.DB.Child("auth", "tokens").Delete(ctx, sa.Token)
	}

	// Cascading delete RoleBindings
	if rbs, err := ListRoleBindingsAll(ctx); err == nil {
		for _, rb := range rbs {
			if rb.ServiceAccountName == name {
				DeleteRoleBinding(ctx, rb.Name)
			}
		}
	}

	_, err = common.DB.Child("auth", "serviceaccounts").Delete(ctx, name)
	return err
}

func GetRole(ctx context.Context, name string) (*Role, error) {
	db := common.DB.Child("auth", "roles")
	data, err := db.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	var role Role
	if err := json.Unmarshal([]byte(data), &role); err != nil {
		return nil, err
	}
	return &role, nil
}

func SaveRole(ctx context.Context, role *Role) error {
	db := common.DB.Child("auth", "roles")
	data, err := json.Marshal(role)
	if err != nil {
		return err
	}
	return db.Put(ctx, role.Name, string(data), kv.TTLKeep)
}

func DeleteRole(ctx context.Context, name string) error {
	// Update RoleBindings instead of cascading delete
	if rbs, err := ListRoleBindingsAll(ctx); err == nil {
		for _, rb := range rbs {
			newRoleNames := make([]string, 0)
			changed := false
			for _, rn := range rb.RoleNames {
				if rn == name {
					changed = true
					continue
				}
				newRoleNames = append(newRoleNames, rn)
			}

			if changed {
				if len(newRoleNames) == 0 {
					// No roles left, delete the binding
					DeleteRoleBinding(ctx, rb.Name)
				} else {
					// Update the binding with remaining roles
					rb.RoleNames = newRoleNames
					SaveRoleBinding(ctx, &rb)
				}
			}
		}
	}

	_, err := common.DB.Child("auth", "roles").Delete(ctx, name)
	return err
}

func GetRoleBinding(ctx context.Context, name string) (*RoleBinding, error) {
	db := common.DB.Child("auth", "rolebindings")
	data, err := db.Get(ctx, name)
	if err != nil {
		return nil, err
	}
	var rb RoleBinding
	if err := json.Unmarshal([]byte(data), &rb); err != nil {
		return nil, err
	}
	return &rb, nil
}

func SaveRoleBinding(ctx context.Context, rb *RoleBinding) error {
	db := common.DB.Child("auth", "rolebindings")
	data, err := json.Marshal(rb)
	if err != nil {
		return err
	}
	return db.Put(ctx, rb.Name, string(data), kv.TTLKeep)
}

func DeleteRoleBinding(ctx context.Context, name string) error {
	_, err := common.DB.Child("auth", "rolebindings").Delete(ctx, name)
	return err
}

func ListServiceAccounts(ctx context.Context, page uint64, pageSize uint, search string) ([]ServiceAccount, int64, error) {
	db := common.DB.Child("auth", "serviceaccounts")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, 0, err
	}
	res := make([]ServiceAccount, 0)
	search = strings.ToLower(search)
	for _, v := range items {
		var sa ServiceAccount
		if err := json.Unmarshal([]byte(v), &sa); err == nil {
			if search == "" || strings.Contains(strings.ToLower(sa.Name), search) {
				res = append(res, sa)
			}
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})

	total := int64(len(res))
	start := int(page) * int(pageSize)
	if start >= len(res) {
		return []ServiceAccount{}, total, nil
	}
	end := start + int(pageSize)
	if end > len(res) {
		end = len(res)
	}

	return res[start:end], total, nil
}

func ListServiceAccountsAll(ctx context.Context) ([]ServiceAccount, error) {
	db := common.DB.Child("auth", "serviceaccounts")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	res := make([]ServiceAccount, 0)
	for _, v := range items {
		var sa ServiceAccount
		if err := json.Unmarshal([]byte(v), &sa); err == nil {
			res = append(res, sa)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res, nil
}

func ListRoles(ctx context.Context, page uint64, pageSize uint, search string) ([]Role, int64, error) {
	db := common.DB.Child("auth", "roles")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, 0, err
	}
	res := make([]Role, 0)
	search = strings.ToLower(search)
	for _, v := range items {
		var role Role
		if err := json.Unmarshal([]byte(v), &role); err == nil {
			if search == "" || strings.Contains(strings.ToLower(role.Name), search) {
				res = append(res, role)
			}
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})

	total := int64(len(res))
	start := int(page) * int(pageSize)
	if start >= len(res) {
		return []Role{}, total, nil
	}
	end := start + int(pageSize)
	if end > len(res) {
		end = len(res)
	}

	return res[start:end], total, nil
}

func ListRolesAll(ctx context.Context) ([]Role, error) {
	db := common.DB.Child("auth", "roles")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	res := make([]Role, 0)
	for _, v := range items {
		var role Role
		if err := json.Unmarshal([]byte(v), &role); err == nil {
			res = append(res, role)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res, nil
}

func ListRoleBindings(ctx context.Context, page uint64, pageSize uint, search string) ([]RoleBinding, int64, error) {
	db := common.DB.Child("auth", "rolebindings")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, 0, err
	}
	res := make([]RoleBinding, 0)
	search = strings.ToLower(search)
	for _, v := range items {
		var rb RoleBinding
		if err := json.Unmarshal([]byte(v), &rb); err == nil {
			if search == "" || strings.Contains(strings.ToLower(rb.Name), search) {
				res = append(res, rb)
			}
		}
	}

	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})

	total := int64(len(res))
	start := int(page) * int(pageSize)
	if start >= len(res) {
		return []RoleBinding{}, total, nil
	}
	end := start + int(pageSize)
	if end > len(res) {
		end = len(res)
	}

	return res[start:end], total, nil
}

func ListRoleBindingsAll(ctx context.Context) ([]RoleBinding, error) {
	db := common.DB.Child("auth", "rolebindings")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	res := make([]RoleBinding, 0)
	for _, v := range items {
		var rb RoleBinding
		if err := json.Unmarshal([]byte(v), &rb); err == nil {
			res = append(res, rb)
		}
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name < res[j].Name
	})
	return res, nil
}

func GetTokenSA(ctx context.Context, token string) (string, error) {
	return common.DB.Child("auth", "tokens").Get(ctx, token)
}

func Authorize(ctx context.Context, saName string, verb string, resource string) (bool, error) {
	rbs, err := ListRoleBindingsAll(ctx)
	if err != nil {
		return false, err
	}

	for _, rb := range rbs {
		if rb.Enabled && rb.ServiceAccountName == saName {
			for _, roleName := range rb.RoleNames {
				role, err := GetRole(ctx, roleName)
				if err != nil {
					continue
				}
				for _, rule := range role.Rules {
					if match(rule.Verbs, verb) && match(rule.Resources, resource) {
						return true, nil
					}
				}
			}
		}
	}

	return false, nil
}

func AuthorizeByToken(ctx context.Context, token string, verb string, resource string) (bool, error) {
	if token == "" {
		return false, nil
	}
	saName, err := GetTokenSA(ctx, token)
	if err != nil || saName == "" {
		return false, nil
	}
	return Authorize(ctx, saName, verb, resource)
}

func match(list []string, item string) bool {
	for _, v := range list {
		if v == "*" || v == item {
			return true
		}
	}
	return false
}
