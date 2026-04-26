package ip

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	ipmodel "homelab/pkg/models/network/ip"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/ip"
	"path/filepath"
	"slices"
	"time"
)

func (s *IPPoolService) CreateGroup(ctx context.Context, group *ipmodel.IPPool) error {
	if group.ID == "" {
		return fmt.Errorf("%w: id is required for IP pool", common.ErrBadRequest)
	}
	if err := requireIPResource(ctx, ipGroupResource(group.ID)); err != nil {
		return err
	}
	group.Status.CreatedAt = time.Now()
	group.Status.UpdatedAt = time.Now()
	err := repo.SavePool(ctx, group)
	commonaudit.FromContext(ctx).Log("CreateIPGroup", group.Meta.Name, "Created", err == nil)
	return err
}

func (s *IPPoolService) UpdateGroup(ctx context.Context, group *ipmodel.IPPool) error {
	if err := requireIPResource(ctx, ipGroupResource(group.ID)); err != nil {
		return err
	}

	current, err := repo.GetPool(ctx, group.ID)
	if err == nil {
		current.Meta = group.Meta
		current.Status.UpdatedAt = time.Now()
		err = repo.SavePool(ctx, current)
	}
	commonaudit.FromContext(ctx).Log("UpdateIPGroup", group.Meta.Name, "Updated", err == nil)
	return err
}

func (s *IPPoolService) DeleteGroup(ctx context.Context, id string) error {
	if err := requireIPResource(ctx, ipGroupResource(id)); err != nil {
		return err
	}

	release, err := s.lockPool(ctx, id)
	if err != nil {
		return err
	}
	defer release()

	old, _ := repo.GetPool(ctx, id)
	// 校验依赖：检查是否有 IPExport 引用了此池
	resExports, err := repo.ScanAllExports(ctx)
	if err != nil {
		return err
	}
	for _, e := range resExports {
		if slices.Contains(e.Meta.GroupIDs, id) {
			return fmt.Errorf("cannot delete group %s: referenced by export %s", id, e.Meta.Name)
		}
	}

	// 校验依赖：检查是否有同步策略引用了此池
	resPolicies, err := repo.ScanAllSyncPolicies(ctx)
	if err != nil {
		return err
	}
	for _, p := range resPolicies {
		if p.Meta.TargetGroupID == id {
			return fmt.Errorf("cannot delete group %s: referenced by sync policy %s", id, p.Meta.Name)
		}
	}

	err = repo.DeletePool(ctx, id)
	if err != nil {
		return err
	}

	// 级联删除 VFS 中的数据文件
	poolPath := filepath.Join(PoolsDir, id+".bin")
	_ = s.deps.FS.Remove(poolPath)
	if s.analysisEngine != nil {
		notifyIPPoolChanged(ctx, id)
	}

	if old != nil {
		commonaudit.FromContext(ctx).Log("DeleteIPGroup", old.Meta.Name, "Deleted", true)
	}
	return nil
}

func (s *IPPoolService) GetGroup(ctx context.Context, id string) (*ipmodel.IPPool, error) {
	if err := requireIPResourceOrGlobal(ctx, ipGroupResource(id)); err != nil {
		return nil, err
	}
	return repo.GetPool(ctx, id)
}

func (s *IPPoolService) LookupGroup(ctx context.Context, id string) (interface{}, error) {
	return repo.GetPool(ctx, id)
}

func (s *IPPoolService) ScanGroups(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[ipmodel.IPPool], error) {
	perms := commonauth.PermissionsFromContext(ctx)
	hasGlobal := perms.IsAllowed(ipResourceBase)
	res, err := repo.ScanPools(ctx, cursor, limit, search)
	if err != nil {
		return nil, err
	}
	if hasGlobal {
		return res, nil
	}
	filtered := make([]ipmodel.IPPool, 0, len(res.Items))
	for _, item := range res.Items {
		if perms.IsAllowed(ipGroupResource(item.ID)) {
			filtered = append(filtered, item)
		}
	}
	res.Items = filtered
	return res, nil
}
