package site

import (
	"context"
	"homelab/pkg/common"
	"strings"

	sitemodel "homelab/pkg/models/network/site"
	"homelab/pkg/models/shared"

	"gopkg.d7z.net/middleware/kv"
)

var (
	groupRepo      *common.ResourceRepository[sitemodel.SiteGroupV1Meta, sitemodel.SiteGroupV1Status]
	exportRepo     *common.ResourceRepository[sitemodel.SiteExportV1Meta, sitemodel.SiteExportV1Status]
	syncPolicyRepo *common.ResourceRepository[sitemodel.SiteSyncPolicyV1Meta, sitemodel.SiteSyncPolicyV1Status]
)

func Configure(db kv.KV) {
	groupRepo = common.NewResourceRepository[sitemodel.SiteGroupV1Meta, sitemodel.SiteGroupV1Status](db, "network", "SiteGroup")
	exportRepo = common.NewResourceRepository[sitemodel.SiteExportV1Meta, sitemodel.SiteExportV1Status](db, "network", "SiteExport")
	syncPolicyRepo = common.NewResourceRepository[sitemodel.SiteSyncPolicyV1Meta, sitemodel.SiteSyncPolicyV1Status](db, "network", "SiteSyncPolicy")
}

// Group Repo
func GetGroup(ctx context.Context, id string) (*sitemodel.SiteGroup, error) {
	return groupRepo.Get(ctx, id)
}

func SaveGroup(ctx context.Context, group *sitemodel.SiteGroup) error {
	return groupRepo.Save(ctx, group)
}

func UpdateGroupStatus(ctx context.Context, id string, apply func(*sitemodel.SiteGroupV1Status)) error {
	return groupRepo.UpdateStatus(ctx, id, apply)
}

func DeleteGroup(ctx context.Context, id string) error {
	return groupRepo.Delete(ctx, id)
}

func ScanGroups(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[sitemodel.SiteGroup], error) {
	search = strings.ToLower(search)
	return groupRepo.List(ctx, cursor, limit, func(g *sitemodel.SiteGroup) bool {
		return search == "" || strings.Contains(strings.ToLower(g.Meta.Name), search) || strings.Contains(strings.ToLower(g.ID), search)
	})
}

// Export Repo
func GetExport(ctx context.Context, id string) (*sitemodel.SiteExport, error) {
	return exportRepo.Get(ctx, id)
}

func SaveExport(ctx context.Context, export *sitemodel.SiteExport) error {
	return exportRepo.Save(ctx, export)
}

func DeleteExport(ctx context.Context, id string) error {
	return exportRepo.Delete(ctx, id)
}

func ScanExports(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[sitemodel.SiteExport], error) {
	search = strings.ToLower(search)
	return exportRepo.List(ctx, cursor, limit, func(e *sitemodel.SiteExport) bool {
		return search == "" || strings.Contains(strings.ToLower(e.Meta.Name), search) || strings.Contains(strings.ToLower(e.ID), search)
	})
}

func ScanAllExports(ctx context.Context) ([]sitemodel.SiteExport, error) {
	return exportRepo.ListAll(ctx)
}

// SyncPolicy Repo
func GetSyncPolicy(ctx context.Context, id string) (*sitemodel.SiteSyncPolicy, error) {
	return syncPolicyRepo.Get(ctx, id)
}

func SaveSyncPolicy(ctx context.Context, policy *sitemodel.SiteSyncPolicy) error {
	return syncPolicyRepo.Save(ctx, policy)
}

func DeleteSyncPolicy(ctx context.Context, id string) error {
	return syncPolicyRepo.Delete(ctx, id)
}

func ScanSyncPolicies(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[sitemodel.SiteSyncPolicy], error) {
	search = strings.ToLower(search)
	return syncPolicyRepo.List(ctx, cursor, limit, func(p *sitemodel.SiteSyncPolicy) bool {
		return search == "" || strings.Contains(strings.ToLower(p.Meta.Name), search) || strings.Contains(strings.ToLower(p.ID), search)
	})
}

func ScanAllSyncPolicies(ctx context.Context) ([]sitemodel.SiteSyncPolicy, error) {
	return syncPolicyRepo.ListAll(ctx)
}
