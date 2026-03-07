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
	"time"

	"github.com/google/uuid"
)

func (s *SitePoolService) CreateGroup(ctx context.Context, group *models.SiteGroup) error {
	group.ID = uuid.NewString()
	resource := "network/site/" + group.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	err := repo.SaveGroup(ctx, group)
	commonaudit.FromContext(ctx).Log("CreateSiteGroup", group.Name, "Created", err == nil)
	return err
}

func (s *SitePoolService) UpdateGroup(ctx context.Context, group *models.SiteGroup) error {
	resource := "network/site/" + group.ID
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
	commonaudit.FromContext(ctx).Log("UpdateSiteGroup", group.Name, "Updated", err == nil)
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
	exports, _, err := repo.ListExports(ctx, 1, 1000, "")
	if err != nil {
		return err
	}
	for _, e := range exports {
		if slices.Contains(e.GroupIDs, id) {
			return fmt.Errorf("cannot delete group %s: referenced by export %s", id, e.Name)
		}
	}

	if err := repo.DeleteGroup(ctx, id); err != nil {
		return err
	}

	poolPath := filepath.Join(PoolsDir, id+".bin")
	_ = common.FS.Remove(poolPath)
	if s.engine != nil {
		notifySitePoolUpdate(ctx, id)
	}

	if old != nil {
		commonaudit.FromContext(ctx).Log("DeleteSiteGroup", old.Name, "Deleted", true)
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

func (s *SitePoolService) ListGroups(ctx context.Context, page, pageSize int, search string) ([]models.SiteGroup, int, error) {
	groups, _, err := repo.ListGroups(ctx, 1, 10000, search)
	if err != nil {
		return nil, 0, err
	}
	var filtered []models.SiteGroup
	perms := commonauth.PermissionsFromContext(ctx)
	for _, g := range groups {
		if perms.IsAllowed("network/site") || perms.IsAllowed("network/site/"+g.ID) {
			filtered = append(filtered, g)
		}
	}
	total := len(filtered)
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []models.SiteGroup{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}
