package site

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/site"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

func (s *SitePoolService) CreateGroup(ctx context.Context, group *models.SiteGroup) error {
	if group.ID == "" {
		return fmt.Errorf("%w: id is required for site group", common.ErrBadRequest)
	}
	resource := "network/site/" + group.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	err := repo.GroupRepo.Cow(ctx, group.ID, func(res *models.Resource[models.SiteGroupV1Meta, models.SiteGroupV1Status]) error {
		res.Meta = group.Meta
		res.Status.CreatedAt = time.Now()
		res.Status.UpdatedAt = time.Now()
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	})

	if err == nil {
		// Update caller object with server-side assigned fields
		updated, _ := repo.GetGroup(ctx, group.ID)
		if updated != nil {
			*group = *updated
		}
	}

	commonaudit.FromContext(ctx).Log("CreateSiteGroup", group.Meta.Name, "Created", err == nil)
	return err
}

func (s *SitePoolService) UpdateGroup(ctx context.Context, group *models.SiteGroup) error {
	resource := "network/site/" + group.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	err := repo.GroupRepo.PatchMeta(ctx, group.ID, group.Generation, func(meta *models.SiteGroupV1Meta) {
		*meta = group.Meta
	})
	commonaudit.FromContext(ctx).Log("UpdateSiteGroup", group.Meta.Name, "Updated", err == nil)
	return err
}

func (s *SitePoolService) DeleteGroup(ctx context.Context, id string) error {
	resource := "network/site/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
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

func (s *SitePoolService) GetGroup(ctx context.Context, id string) (*models.SiteGroup, error) {
	resource := "network/site/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) && !commonauth.PermissionsFromContext(ctx).IsAllowed("network/site") {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	return repo.GetGroup(ctx, id)
}

func (s *SitePoolService) ScanGroups(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.SiteGroup], error) {
	perms := commonauth.PermissionsFromContext(ctx)
	hasGlobal := perms.IsAllowed("network/site")

	search = strings.ToLower(search)
	return repo.GroupRepo.List(ctx, cursor, limit, func(g *models.Resource[models.SiteGroupV1Meta, models.SiteGroupV1Status]) bool {
		if !hasGlobal && !perms.IsAllowed("network/site/"+g.ID) {
			return false
		}
		return search == "" || strings.Contains(strings.ToLower(g.Meta.Name), search) || strings.Contains(strings.ToLower(g.ID), search)
	})
}
