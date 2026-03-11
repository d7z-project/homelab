package rbac

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

var (
	roleCache *lru.Cache[string, *models.Role]

	RoleRepo           = common.NewBaseRepository[models.RoleV1Meta, models.RoleV1Status]("auth", "roles")
	BindingRepo        = common.NewBaseRepository[models.RoleBindingV1Meta, models.RoleBindingV1Status]("auth", "rolebindings")
	ServiceAccountRepo = common.NewBaseRepository[models.ServiceAccountV1Meta, models.ServiceAccountV1Status]("auth", "serviceaccounts")
)

func init() {
	roleCache, _ = lru.New[string, *models.Role](1024)
}

func ClearCache() {
	roleCache.Purge()
}

func InvalidateCache(roleID string) {
	if roleID != "" {
		roleCache.Remove(roleID)
	}
}

// Legacy compatibility helpers

func GetServiceAccount(ctx context.Context, id string) (*models.ServiceAccount, error) {
	return ServiceAccountRepo.Get(ctx, id)
}

func SaveServiceAccount(ctx context.Context, sa *models.ServiceAccount) error {
	return ServiceAccountRepo.Save(ctx, sa)
}

func DeleteServiceAccount(ctx context.Context, id string) error {
	return ServiceAccountRepo.Delete(ctx, id)
}

func ScanServiceAccounts(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.ServiceAccount], error) {
	return ServiceAccountRepo.List(ctx, cursor, limit, nil)
}

func GetRole(ctx context.Context, id string) (*models.Role, error) {
	if val, ok := roleCache.Get(id); ok {
		return val, nil
	}
	role, err := RoleRepo.Get(ctx, id)
	if err == nil {
		roleCache.Add(id, role)
	}
	return role, err
}

func SaveRole(ctx context.Context, role *models.Role) error {
	err := RoleRepo.Save(ctx, role)
	if err == nil {
		roleCache.Add(role.ID, role)
		InvalidateCache("")
	}
	return err
}

func DeleteRole(ctx context.Context, id string) error {
	err := RoleRepo.Delete(ctx, id)
	if err == nil {
		InvalidateCache(id)
	}
	return err
}

func ScanRoles(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.Role], error) {
	return RoleRepo.List(ctx, cursor, limit, nil)
}

func GetRoleBinding(ctx context.Context, id string) (*models.RoleBinding, error) {
	return BindingRepo.Get(ctx, id)
}

func SaveRoleBinding(ctx context.Context, rb *models.RoleBinding) error {
	err := BindingRepo.Save(ctx, rb)
	if err == nil {
		InvalidateCache("")
	}
	return err
}

func DeleteRoleBinding(ctx context.Context, id string) error {
	err := BindingRepo.Delete(ctx, id)
	if err == nil {
		InvalidateCache("")
	}
	return err
}

func ScanRoleBindings(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.RoleBinding], error) {
	return BindingRepo.List(ctx, cursor, limit, nil)
}

func ScanAllRoleBindings(ctx context.Context) ([]models.RoleBinding, error) {
	resp, err := BindingRepo.List(ctx, "", 100000, nil)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func GetLastModified() time.Time {
	return time.Time{} // Simplified for now as BaseRepository handles consistency
}
