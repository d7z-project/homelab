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
	"strings"
	"time"
)

func (s *IPPoolService) CreateGroup(ctx context.Context, group *models.IPPool) error {
	if group.ID == "" {
		return fmt.Errorf("%w: id is required for IP pool", common.ErrBadRequest)
	}
	resource := "network/ip/" + group.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	err := repo.PoolRepo.Cow(ctx, group.ID, func(res *models.IPPool) error {
		res.Meta = group.Meta
		res.Status.CreatedAt = time.Now()
		res.Status.UpdatedAt = time.Now()
		res.Generation = 1
		res.ResourceVersion = 1

		group.Status = res.Status
		group.Generation = res.Generation
		group.ResourceVersion = res.ResourceVersion
		return nil
	})
	commonaudit.FromContext(ctx).Log("CreateIPGroup", group.Meta.Name, "Created", err == nil)
	return err
}

func (s *IPPoolService) UpdateGroup(ctx context.Context, group *models.IPPool) error {
	resource := "network/ip/" + group.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	err := repo.PoolRepo.PatchMeta(ctx, group.ID, group.Generation, func(m *models.IPPoolV1Meta) {
		*m = group.Meta
	})
	commonaudit.FromContext(ctx).Log("UpdateIPGroup", group.Meta.Name, "Updated", err == nil)
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

	old, _ := repo.PoolRepo.Get(ctx, id)
	// 校验依赖：检查是否有 IPExport 引用了此池
	resExports, err := repo.ExportRepo.List(ctx, "", 1000, nil)
	if err != nil {
		return err
	}
	for _, e := range resExports.Items {
		if slices.Contains(e.Meta.GroupIDs, id) {
			return fmt.Errorf("cannot delete group %s: referenced by export %s", id, e.Meta.Name)
		}
	}

	// 校验依赖：检查是否有同步策略引用了此池
	resPolicies, err := repo.SyncPolicyRepo.List(ctx, "", 1000, nil)
	if err != nil {
		return err
	}
	for _, p := range resPolicies.Items {
		if p.Meta.TargetGroupID == id {
			return fmt.Errorf("cannot delete group %s: referenced by sync policy %s", id, p.Meta.Name)
		}
	}

	err = repo.PoolRepo.Delete(ctx, id)
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
		commonaudit.FromContext(ctx).Log("DeleteIPGroup", old.Meta.Name, "Deleted", true)
	}
	return nil
}

func (s *IPPoolService) GetGroup(ctx context.Context, id string) (*models.IPPool, error) {
	resource := "network/ip/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) && !commonauth.PermissionsFromContext(ctx).IsAllowed("network/ip") {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	return repo.PoolRepo.Get(ctx, id)
}

func (s *IPPoolService) LookupGroup(ctx context.Context, id string) (interface{}, error) {
	return repo.PoolRepo.Get(ctx, id)
}

func (s *IPPoolService) ScanGroups(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.IPPool], error) {
	perms := commonauth.PermissionsFromContext(ctx)
	hasGlobal := perms.IsAllowed("network/ip")
	search = strings.ToLower(search)

	filter := func(p *models.IPPool) bool {
		if !hasGlobal && !perms.IsAllowed("network/ip/"+p.ID) {
			return false
		}
		if search != "" {
			return strings.Contains(strings.ToLower(p.Meta.Name), search) || strings.Contains(strings.ToLower(p.ID), search)
		}
		return true
	}

	return repo.PoolRepo.List(ctx, cursor, limit, filter)
}
