package site

import (
	"context"
	"homelab/pkg/common"
	"strings"

	sitemodel "homelab/pkg/models/network/site"
	"homelab/pkg/models/shared"
)

var (
	GroupRepo      = common.NewBaseRepository[sitemodel.SiteGroupV1Meta, sitemodel.SiteGroupV1Status]("network", "SiteGroup")
	ExportRepo     = common.NewBaseRepository[sitemodel.SiteExportV1Meta, sitemodel.SiteExportV1Status]("network", "SiteExport")
	SyncPolicyRepo = common.NewBaseRepository[sitemodel.SiteSyncPolicyV1Meta, sitemodel.SiteSyncPolicyV1Status]("network", "SiteSyncPolicy")
)

// Group Repo
func GetGroup(ctx context.Context, id string) (*sitemodel.SiteGroup, error) {
	return GroupRepo.Get(ctx, id)
}

func SaveGroup(ctx context.Context, group *sitemodel.SiteGroup) error {
	return GroupRepo.Save(ctx, group)
}

func DeleteGroup(ctx context.Context, id string) error {
	return GroupRepo.Delete(ctx, id)
}

func ScanGroups(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[sitemodel.SiteGroup], error) {
	search = strings.ToLower(search)
	return GroupRepo.List(ctx, cursor, limit, func(g *sitemodel.SiteGroup) bool {
		return search == "" || strings.Contains(strings.ToLower(g.Meta.Name), search) || strings.Contains(strings.ToLower(g.ID), search)
	})
}

// Export Repo
func GetExport(ctx context.Context, id string) (*sitemodel.SiteExport, error) {
	return ExportRepo.Get(ctx, id)
}

func SaveExport(ctx context.Context, export *sitemodel.SiteExport) error {
	return ExportRepo.Save(ctx, export)
}

func DeleteExport(ctx context.Context, id string) error {
	return ExportRepo.Delete(ctx, id)
}

func ScanExports(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[sitemodel.SiteExport], error) {
	search = strings.ToLower(search)
	return ExportRepo.List(ctx, cursor, limit, func(e *sitemodel.SiteExport) bool {
		return search == "" || strings.Contains(strings.ToLower(e.Meta.Name), search) || strings.Contains(strings.ToLower(e.ID), search)
	})
}

// SyncPolicy Repo
func GetSyncPolicy(ctx context.Context, id string) (*sitemodel.SiteSyncPolicy, error) {
	return SyncPolicyRepo.Get(ctx, id)
}

func SaveSyncPolicy(ctx context.Context, policy *sitemodel.SiteSyncPolicy) error {
	return SyncPolicyRepo.Save(ctx, policy)
}

func DeleteSyncPolicy(ctx context.Context, id string) error {
	return SyncPolicyRepo.Delete(ctx, id)
}

func ScanSyncPolicies(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[sitemodel.SiteSyncPolicy], error) {
	search = strings.ToLower(search)
	return SyncPolicyRepo.List(ctx, cursor, limit, func(p *sitemodel.SiteSyncPolicy) bool {
		return search == "" || strings.Contains(strings.ToLower(p.Meta.Name), search) || strings.Contains(strings.ToLower(p.ID), search)
	})
}
