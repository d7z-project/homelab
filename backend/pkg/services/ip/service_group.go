package ip

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
	"path/filepath"
	"slices"
	"time"

	"github.com/google/uuid"
)

func (s *IPPoolService) CreateGroup(ctx context.Context, group *models.IPGroup) error {
	if group.ID == "" {
		group.ID = uuid.NewString()
	}
	resource := "network/ip/" + group.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	err := repo.SaveGroup(ctx, group)
	commonaudit.FromContext(ctx).Log("CreateIPGroup", group.Name, "Created", err == nil)
	return err
}

func (s *IPPoolService) UpdateGroup(ctx context.Context, group *models.IPGroup) error {
	resource := "network/ip/" + group.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	old, err := repo.GetGroup(ctx, group.ID)
	if err != nil {
		return err
	}
	group.CreatedAt = old.CreatedAt
	group.UpdatedAt = time.Now()
	err = repo.SaveGroup(ctx, group)
	commonaudit.FromContext(ctx).Log("UpdateIPGroup", group.Name, "Updated", err == nil)
	return err
}

func (s *IPPoolService) DeleteGroup(ctx context.Context, id string) error {
	resource := "network/ip/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	release, err := s.lockPool(ctx, id)
	if err != nil {
		return err
	}
	defer release()

	old, _ := repo.GetGroup(ctx, id)
	// 校验依赖：检查是否有 IPExport 引用了此池
	resExports, err := repo.ScanExports(ctx, "", 1000, "")
	if err != nil {
		return err
	}
	for _, e := range resExports.Items {
		if slices.Contains(e.GroupIDs, id) {
			return fmt.Errorf("cannot delete group %s: referenced by export %s", id, e.Name)
		}
	}

	// 校验依赖：检查是否有同步策略引用了此池
	resPolicies, err := repo.ScanSyncPolicies(ctx, "", 1000, "")
	if err != nil {
		return err
	}
	for _, p := range resPolicies.Items {
		if p.TargetGroupID == id {
			return fmt.Errorf("cannot delete group %s: referenced by sync policy %s", id, p.Name)
		}
	}

	// 删除 DB 记录
	err = repo.DeleteGroup(ctx, id)
	if err != nil {
		return err
	}

	// 级联删除 VFS 中的数据文件
	poolPath := filepath.Join(PoolsDir, id+".bin")
	_ = common.FS.Remove(poolPath)
	if s.analysisEngine != nil {
		notifyIPPoolChanged(ctx, id)
	}

	if old != nil {
		commonaudit.FromContext(ctx).Log("DeleteIPGroup", old.Name, "Deleted", true)
	}
	return nil
}

func (s *IPPoolService) GetGroup(ctx context.Context, id string) (*models.IPGroup, error) {
	resource := "network/ip/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) && !commonauth.PermissionsFromContext(ctx).IsAllowed("network/ip") {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	return repo.GetGroup(ctx, id)
}

func (s *IPPoolService) LookupGroup(ctx context.Context, id string) (interface{}, error) {
	return repo.GetGroup(ctx, id)
}

func (s *IPPoolService) ScanGroups(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.IPGroup], error) {
	// Use a larger limit for repo scan to account for permission filtering
	res, err := repo.ScanGroups(ctx, cursor, limit*2, search)
	if err != nil {
		return nil, err
	}

	var filtered []models.IPGroup
	perms := commonauth.PermissionsFromContext(ctx)
	for _, g := range res.Items {
		if perms.IsAllowed("network/ip") || perms.IsAllowed("network/ip/"+g.ID) {
			filtered = append(filtered, g)
		}
		if len(filtered) >= limit {
			return &models.PaginationResponse[models.IPGroup]{
				Items:      filtered,
				NextCursor: res.NextCursor,
				HasMore:    res.HasMore,
			}, nil
		}
	}

	return &models.PaginationResponse[models.IPGroup]{
		Items:      filtered,
		NextCursor: res.NextCursor,
		HasMore:    res.HasMore,
	}, nil
}
