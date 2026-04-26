package rbac

import (
	"context"
	"homelab/pkg/common"
	"strings"

	rbacmodel "homelab/pkg/models/core/rbac"
	"homelab/pkg/models/shared"

	lru "github.com/hashicorp/golang-lru/v2"
)

var (
	roleCache *lru.Cache[string, *rbacmodel.Role]

	roleRepo           = common.NewBaseRepository[rbacmodel.RoleV1Meta, rbacmodel.RoleV1Status]("auth", "roles")
	bindingRepo        = common.NewBaseRepository[rbacmodel.RoleBindingV1Meta, rbacmodel.RoleBindingV1Status]("auth", "rolebindings")
	serviceAccountRepo = common.NewBaseRepository[rbacmodel.ServiceAccountV1Meta, rbacmodel.ServiceAccountV1Status]("auth", "serviceaccounts")
)

func init() {
	roleCache, _ = lru.New[string, *rbacmodel.Role](1024)
}

func ClearCache() {
	roleCache.Purge()
}

func InvalidateCache(roleID string) {
	if roleID != "" {
		roleCache.Remove(roleID)
	}
}

func ScanServiceAccounts(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[rbacmodel.ServiceAccount], error) {
	search = strings.ToLower(search)
	return serviceAccountRepo.List(ctx, cursor, limit, func(sa *rbacmodel.ServiceAccount) bool {
		return search == "" || strings.Contains(strings.ToLower(sa.Meta.Name), search) || strings.Contains(strings.ToLower(sa.ID), search)
	})
}

func GetServiceAccount(ctx context.Context, id string) (*rbacmodel.ServiceAccount, error) {
	return serviceAccountRepo.Get(ctx, id)
}

func SaveServiceAccount(ctx context.Context, sa *rbacmodel.ServiceAccount) error {
	return serviceAccountRepo.Save(ctx, sa)
}

func DeleteServiceAccount(ctx context.Context, id string) error {
	return serviceAccountRepo.Delete(ctx, id)
}

func UpdateServiceAccountStatus(ctx context.Context, id string, apply func(*rbacmodel.ServiceAccountV1Status)) error {
	return serviceAccountRepo.UpdateStatus(ctx, id, apply)
}

func GetCachedRole(ctx context.Context, id string) (*rbacmodel.Role, error) {
	if val, ok := roleCache.Get(id); ok {
		return val, nil
	}
	role, err := roleRepo.Get(ctx, id)
	if err == nil {
		roleCache.Add(id, role)
	}
	return role, err
}

func ScanRoles(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[rbacmodel.Role], error) {
	search = strings.ToLower(search)
	return roleRepo.List(ctx, cursor, limit, func(role *rbacmodel.Role) bool {
		return search == "" || strings.Contains(strings.ToLower(role.Meta.Name), search) || strings.Contains(strings.ToLower(role.ID), search)
	})
}

func GetRole(ctx context.Context, id string) (*rbacmodel.Role, error) {
	return roleRepo.Get(ctx, id)
}

func SaveRole(ctx context.Context, role *rbacmodel.Role) error {
	return roleRepo.Save(ctx, role)
}

func DeleteRole(ctx context.Context, id string) error {
	return roleRepo.Delete(ctx, id)
}

func ScanRoleBindings(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[rbacmodel.RoleBinding], error) {
	search = strings.ToLower(search)
	return bindingRepo.List(ctx, cursor, limit, func(binding *rbacmodel.RoleBinding) bool {
		return search == "" || strings.Contains(strings.ToLower(binding.ID), search) || strings.Contains(strings.ToLower(binding.Meta.ServiceAccountID), search)
	})
}

func GetRoleBinding(ctx context.Context, id string) (*rbacmodel.RoleBinding, error) {
	return bindingRepo.Get(ctx, id)
}

func SaveRoleBinding(ctx context.Context, binding *rbacmodel.RoleBinding) error {
	return bindingRepo.Save(ctx, binding)
}

func DeleteRoleBinding(ctx context.Context, id string) error {
	return bindingRepo.Delete(ctx, id)
}

func ScanAllRoleBindings(ctx context.Context) ([]rbacmodel.RoleBinding, error) {
	return bindingRepo.ListAll(ctx)
}
