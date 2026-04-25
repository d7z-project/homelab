package site

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	sitemodel "homelab/pkg/models/network/site"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/site"
	ruleservice "homelab/pkg/services/rules"
	"path/filepath"
	"slices"
	"time"
)

func (s *SitePoolService) CreateGroup(ctx context.Context, group *sitemodel.SiteGroup) error {
	if group.ID == "" {
		return fmt.Errorf("%w: id is required for site group", common.ErrBadRequest)
	}
	if err := requireSiteResource(ctx, siteGroupResource(group.ID)); err != nil {
		return err
	}

	err := ruleservice.CreateAndLoad(ctx, repo.GroupRepo, group, func(res *shared.Resource[sitemodel.SiteGroupV1Meta, sitemodel.SiteGroupV1Status]) error {
		res.Meta = group.Meta
		res.Status.CreatedAt = time.Now()
		res.Status.UpdatedAt = time.Now()
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	})

	commonaudit.FromContext(ctx).Log("CreateSiteGroup", group.Meta.Name, "Created", err == nil)
	return err
}

func (s *SitePoolService) UpdateGroup(ctx context.Context, group *sitemodel.SiteGroup) error {
	if err := requireSiteResource(ctx, siteGroupResource(group.ID)); err != nil {
		return err
	}

	err := ruleservice.ReplaceMeta(ctx, repo.GroupRepo, group)
	commonaudit.FromContext(ctx).Log("UpdateSiteGroup", group.Meta.Name, "Updated", err == nil)
	return err
}

func (s *SitePoolService) DeleteGroup(ctx context.Context, id string) error {
	if err := requireSiteResource(ctx, siteGroupResource(id)); err != nil {
		return err
	}

	release, err := s.lockPool(ctx, id)
	if err != nil {
		return err
	}
	defer release()

	old, _ := repo.GetGroup(ctx, id)
	exportsResp, err := repo.ScanExports(ctx, "", 1000, "")
	if err != nil {
		return err
	}
	for _, e := range exportsResp.Items {
		if slices.Contains(e.Meta.GroupIDs, id) {
			return fmt.Errorf("cannot delete group %s: referenced by export %s", id, e.Meta.Name)
		}
	}

	if err := repo.DeleteGroup(ctx, id); err != nil {
		return err
	}

	poolPath := filepath.Join(PoolsDir, id+".bin")
	_ = common.FS.Remove(poolPath)
	if s.engine != nil {
		notifySitePoolChanged(ctx, id)
	}

	if old != nil {
		commonaudit.FromContext(ctx).Log("DeleteSiteGroup", old.Meta.Name, "Deleted", true)
	}
	return nil
}

func (s *SitePoolService) GetGroup(ctx context.Context, id string) (*sitemodel.SiteGroup, error) {
	if err := requireSiteResourceOrGlobal(ctx, siteGroupResource(id)); err != nil {
		return nil, err
	}
	return repo.GetGroup(ctx, id)
}

func (s *SitePoolService) ScanGroups(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[sitemodel.SiteGroup], error) {
	perms := commonauth.PermissionsFromContext(ctx)
	hasGlobal := perms.IsAllowed(siteResourceBase)
	return ruleservice.ScanBySearch(ctx, repo.GroupRepo, cursor, limit, search, func(g *sitemodel.SiteGroup) bool {
		return hasGlobal || perms.IsAllowed(siteGroupResource(g.ID))
	}, func(meta *sitemodel.SiteGroupV1Meta) string {
		return meta.Name
	})
}
