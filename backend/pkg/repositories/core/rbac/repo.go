package rbac

import (
	"context"
	"homelab/pkg/common"

	rbacmodel "homelab/pkg/models/core/rbac"
	"homelab/pkg/models/shared"

	lru "github.com/hashicorp/golang-lru/v2"
)

var (
	roleCache *lru.Cache[string, *rbacmodel.Role]

	RoleRepo           = common.NewBaseRepository[rbacmodel.RoleV1Meta, rbacmodel.RoleV1Status]("auth", "roles")
	BindingRepo        = common.NewBaseRepository[rbacmodel.RoleBindingV1Meta, rbacmodel.RoleBindingV1Status]("auth", "rolebindings")
	ServiceAccountRepo = common.NewBaseRepository[rbacmodel.ServiceAccountV1Meta, rbacmodel.ServiceAccountV1Status]("auth", "serviceaccounts")
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
	return ServiceAccountRepo.List(ctx, cursor, limit, nil)
}

func GetCachedRole(ctx context.Context, id string) (*rbacmodel.Role, error) {
	if val, ok := roleCache.Get(id); ok {
		return val, nil
	}
	role, err := RoleRepo.Get(ctx, id)
	if err == nil {
		roleCache.Add(id, role)
	}
	return role, err
}

func ScanRoles(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[rbacmodel.Role], error) {
	return RoleRepo.List(ctx, cursor, limit, nil)
}

func ScanRoleBindings(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[rbacmodel.RoleBinding], error) {
	return BindingRepo.List(ctx, cursor, limit, nil)
}

func ScanAllRoleBindings(ctx context.Context) ([]rbacmodel.RoleBinding, error) {
	resp, err := BindingRepo.List(ctx, "", 100000, nil)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}
