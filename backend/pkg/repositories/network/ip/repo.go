package ip

import (
	"context"
	"homelab/pkg/common"
	"strings"

	ipmodel "homelab/pkg/models/network/ip"
	"homelab/pkg/models/shared"
)

var poolRepo = common.NewResourceRepository[ipmodel.IPPoolV1Meta, ipmodel.IPPoolV1Status]("network", "IPPool")
var exportRepo = common.NewResourceRepository[ipmodel.IPExportV1Meta, ipmodel.IPExportV1Status]("network", "IPExport")
var syncPolicyRepo = common.NewResourceRepository[ipmodel.IPSyncPolicyV1Meta, ipmodel.IPSyncPolicyV1Status]("network", "IPSyncPolicy")

func GetSyncPolicy(ctx context.Context, id string) (*ipmodel.IPSyncPolicy, error) {
	return syncPolicyRepo.Get(ctx, id)
}

func GetPool(ctx context.Context, id string) (*ipmodel.IPPool, error) {
	return poolRepo.Get(ctx, id)
}

func SavePool(ctx context.Context, pool *ipmodel.IPPool) error {
	return poolRepo.Save(ctx, pool)
}

func UpdatePoolStatus(ctx context.Context, id string, apply func(*ipmodel.IPPoolV1Status)) error {
	return poolRepo.UpdateStatus(ctx, id, apply)
}

func DeletePool(ctx context.Context, id string) error {
	return poolRepo.Delete(ctx, id)
}

func ScanPools(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[ipmodel.IPPool], error) {
	search = strings.ToLower(search)
	return poolRepo.List(ctx, cursor, limit, func(pool *ipmodel.IPPool) bool {
		return search == "" || strings.Contains(strings.ToLower(pool.Meta.Name), search) || strings.Contains(strings.ToLower(pool.ID), search)
	})
}

func ScanAllPools(ctx context.Context) ([]ipmodel.IPPool, error) {
	return poolRepo.ListAll(ctx)
}

func GetExport(ctx context.Context, id string) (*ipmodel.IPExport, error) {
	return exportRepo.Get(ctx, id)
}

func SaveExport(ctx context.Context, export *ipmodel.IPExport) error {
	return exportRepo.Save(ctx, export)
}

func DeleteExport(ctx context.Context, id string) error {
	return exportRepo.Delete(ctx, id)
}

func ScanExports(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[ipmodel.IPExport], error) {
	search = strings.ToLower(search)
	return exportRepo.List(ctx, cursor, limit, func(export *ipmodel.IPExport) bool {
		return search == "" || strings.Contains(strings.ToLower(export.Meta.Name), search) || strings.Contains(strings.ToLower(export.ID), search)
	})
}

func ScanAllExports(ctx context.Context) ([]ipmodel.IPExport, error) {
	return exportRepo.ListAll(ctx)
}

func SaveSyncPolicy(ctx context.Context, p *ipmodel.IPSyncPolicy) error {
	return syncPolicyRepo.Save(ctx, p)
}

func UpdateSyncPolicyStatus(ctx context.Context, id string, apply func(*ipmodel.IPSyncPolicyV1Status)) error {
	return syncPolicyRepo.UpdateStatus(ctx, id, apply)
}

func DeleteSyncPolicy(ctx context.Context, id string) error {
	return syncPolicyRepo.Delete(ctx, id)
}

func ScanSyncPolicies(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[ipmodel.IPSyncPolicy], error) {
	search = strings.ToLower(search)
	return syncPolicyRepo.List(ctx, cursor, limit, func(policy *ipmodel.IPSyncPolicy) bool {
		return search == "" || strings.Contains(strings.ToLower(policy.Meta.Name), search) || strings.Contains(strings.ToLower(policy.ID), search)
	})
}

func ScanAllSyncPolicies(ctx context.Context) ([]ipmodel.IPSyncPolicy, error) {
	return syncPolicyRepo.ListAll(ctx)
}
