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
	ruleservice "homelab/pkg/services/rules"
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
	err := ruleservice.CreateAndLoad(ctx, repo.PoolRepo, group, func(res *ipmodel.IPPool) error {
		res.Meta = group.Meta
		res.Status.CreatedAt = time.Now()
		res.Status.UpdatedAt = time.Now()
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	})
	commonaudit.FromContext(ctx).Log("CreateIPGroup", group.Meta.Name, "Created", err == nil)
	return err
}

func (s *IPPoolService) UpdateGroup(ctx context.Context, group *ipmodel.IPPool) error {
	if err := requireIPResource(ctx, ipGroupResource(group.ID)); err != nil {
		return err
	}

	err := ruleservice.ReplaceMeta(ctx, repo.PoolRepo, group)
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

func (s *IPPoolService) GetGroup(ctx context.Context, id string) (*ipmodel.IPPool, error) {
	if err := requireIPResourceOrGlobal(ctx, ipGroupResource(id)); err != nil {
		return nil, err
	}
	return repo.PoolRepo.Get(ctx, id)
}

func (s *IPPoolService) LookupGroup(ctx context.Context, id string) (interface{}, error) {
	return repo.PoolRepo.Get(ctx, id)
}

func (s *IPPoolService) ScanGroups(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[ipmodel.IPPool], error) {
	perms := commonauth.PermissionsFromContext(ctx)
	hasGlobal := perms.IsAllowed(ipResourceBase)
	return ruleservice.ScanBySearch(ctx, repo.PoolRepo, cursor, limit, search, func(p *ipmodel.IPPool) bool {
		return hasGlobal || perms.IsAllowed(ipGroupResource(p.ID))
	}, func(meta *ipmodel.IPPoolV1Meta) string {
		return meta.Name
	})
}
