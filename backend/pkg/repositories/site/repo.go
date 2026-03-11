package site

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"
)

var (
	GroupRepo      = common.NewBaseRepository[models.SiteGroupV1Meta, models.SiteGroupV1Status]("network", "SiteGroup")
	ExportRepo     = common.NewBaseRepository[models.SiteExportV1Meta, models.SiteExportV1Status]("network", "SiteExport")
	SyncPolicyRepo = common.NewBaseRepository[models.SiteSyncPolicyV1Meta, models.SiteSyncPolicyV1Status]("network", "SiteSyncPolicy")
)

// Group Repo
func GetGroup(ctx context.Context, id string) (*models.SiteGroup, error) {
	return GroupRepo.Get(ctx, id)
}

func SaveGroup(ctx context.Context, group *models.SiteGroup) error {
	return GroupRepo.Save(ctx, group)
}

func DeleteGroup(ctx context.Context, id string) error {
	return GroupRepo.Delete(ctx, id)
}

func ScanGroups(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.SiteGroup], error) {
	search = strings.ToLower(search)
	return GroupRepo.List(ctx, cursor, limit, func(g *models.SiteGroup) bool {
		return search == "" || strings.Contains(strings.ToLower(g.Meta.Name), search) || strings.Contains(strings.ToLower(g.ID), search)
	})
}

// Export Repo
func GetExport(ctx context.Context, id string) (*models.SiteExport, error) {
	return ExportRepo.Get(ctx, id)
}

func SaveExport(ctx context.Context, export *models.SiteExport) error {
	return ExportRepo.Save(ctx, export)
}

func DeleteExport(ctx context.Context, id string) error {
	return ExportRepo.Delete(ctx, id)
}

func ScanExports(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.SiteExport], error) {
	search = strings.ToLower(search)
	return ExportRepo.List(ctx, cursor, limit, func(e *models.SiteExport) bool {
		return search == "" || strings.Contains(strings.ToLower(e.Meta.Name), search) || strings.Contains(strings.ToLower(e.ID), search)
	})
}

// SyncPolicy Repo
func GetSyncPolicy(ctx context.Context, id string) (*models.SiteSyncPolicy, error) {
	return SyncPolicyRepo.Get(ctx, id)
}

func SaveSyncPolicy(ctx context.Context, policy *models.SiteSyncPolicy) error {
	return SyncPolicyRepo.Save(ctx, policy)
}

func DeleteSyncPolicy(ctx context.Context, id string) error {
	return SyncPolicyRepo.Delete(ctx, id)
}

func ScanSyncPolicies(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.SiteSyncPolicy], error) {
	search = strings.ToLower(search)
	return SyncPolicyRepo.List(ctx, cursor, limit, func(p *models.SiteSyncPolicy) bool {
		return search == "" || strings.Contains(strings.ToLower(p.Meta.Name), search) || strings.Contains(strings.ToLower(p.ID), search)
	})
}
