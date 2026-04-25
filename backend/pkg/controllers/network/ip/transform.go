package ip

import (
	apiv1 "homelab/pkg/apis/network/ip/v1"
	networkcommon "homelab/pkg/models/network/common"
	ipmodel "homelab/pkg/models/network/ip"
	"homelab/pkg/models/shared"
)

func toModelPool(api apiv1.Pool) ipmodel.IPPool {
	return ipmodel.IPPool{
		ID: api.ID,
		Meta: ipmodel.IPPoolV1Meta{
			Name:        api.Meta.Name,
			Description: api.Meta.Description,
		},
		Status: ipmodel.IPPoolV1Status{
			Checksum:   api.Status.Checksum,
			EntryCount: api.Status.EntryCount,
			CreatedAt:  api.Status.CreatedAt,
			UpdatedAt:  api.Status.UpdatedAt,
		},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPIPool(model ipmodel.IPPool) apiv1.Pool {
	return apiv1.Pool{
		ID: model.ID,
		Meta: apiv1.PoolMeta{
			Name:        model.Meta.Name,
			Description: model.Meta.Description,
		},
		Status: apiv1.PoolStatus{
			Checksum:   model.Status.Checksum,
			EntryCount: model.Status.EntryCount,
			CreatedAt:  model.Status.CreatedAt,
			UpdatedAt:  model.Status.UpdatedAt,
		},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func toModelExport(api apiv1.Export) ipmodel.IPExport {
	return ipmodel.IPExport{
		ID: api.ID,
		Meta: ipmodel.IPExportV1Meta{
			Name:        api.Meta.Name,
			Description: api.Meta.Description,
			Rule:        api.Meta.Rule,
			GroupIDs:    append([]string(nil), api.Meta.GroupIDs...),
		},
		Status: ipmodel.IPExportV1Status{
			CreatedAt: api.Status.CreatedAt,
			UpdatedAt: api.Status.UpdatedAt,
		},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPIExport(model ipmodel.IPExport) apiv1.Export {
	return apiv1.Export{
		ID: model.ID,
		Meta: apiv1.ExportMeta{
			Name:        model.Meta.Name,
			Description: model.Meta.Description,
			Rule:        model.Meta.Rule,
			GroupIDs:    append([]string(nil), model.Meta.GroupIDs...),
		},
		Status: apiv1.ExportStatus{
			CreatedAt: model.Status.CreatedAt,
			UpdatedAt: model.Status.UpdatedAt,
		},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func toModelSyncPolicy(api apiv1.SyncPolicy) ipmodel.IPSyncPolicy {
	return ipmodel.IPSyncPolicy{
		ID: api.ID,
		Meta: ipmodel.IPSyncPolicyV1Meta{
			Name:          api.Meta.Name,
			Description:   api.Meta.Description,
			SourceURL:     api.Meta.SourceURL,
			Format:        api.Meta.Format,
			Mode:          api.Meta.Mode,
			Config:        api.Meta.Config,
			TargetGroupID: api.Meta.TargetGroupID,
			Cron:          api.Meta.Cron,
			Enabled:       api.Meta.Enabled,
		},
		Status: ipmodel.IPSyncPolicyV1Status{
			CreatedAt:    api.Status.CreatedAt,
			UpdatedAt:    api.Status.UpdatedAt,
			LastRunAt:    api.Status.LastRunAt,
			LastStatus:   api.Status.LastStatus,
			Progress:     api.Status.Progress,
			ErrorMessage: api.Status.ErrorMessage,
		},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPISyncPolicy(model ipmodel.IPSyncPolicy) apiv1.SyncPolicy {
	return apiv1.SyncPolicy{
		ID: model.ID,
		Meta: apiv1.SyncPolicyMeta{
			Name:          model.Meta.Name,
			Description:   model.Meta.Description,
			SourceURL:     model.Meta.SourceURL,
			Format:        model.Meta.Format,
			Mode:          model.Meta.Mode,
			Config:        model.Meta.Config,
			TargetGroupID: model.Meta.TargetGroupID,
			Cron:          model.Meta.Cron,
			Enabled:       model.Meta.Enabled,
		},
		Status: apiv1.SyncPolicyStatus{
			CreatedAt:    model.Status.CreatedAt,
			UpdatedAt:    model.Status.UpdatedAt,
			LastRunAt:    model.Status.LastRunAt,
			LastStatus:   model.Status.LastStatus,
			Progress:     model.Status.Progress,
			ErrorMessage: model.Status.ErrorMessage,
		},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func toModelPoolEntryRequest(api apiv1.PoolEntryRequest) ipmodel.IPPoolEntryRequest {
	return ipmodel.IPPoolEntryRequest{CIDR: api.CIDR, OldTags: append([]string(nil), api.OldTags...), NewTags: append([]string(nil), api.NewTags...)}
}

func toModelExportPreviewRequest(api apiv1.ExportPreviewRequest) ipmodel.IPExportPreviewRequest {
	return ipmodel.IPExportPreviewRequest{Rule: api.Rule, GroupIDs: append([]string(nil), api.GroupIDs...)}
}

func toAPIPoolEntry(model ipmodel.IPPoolEntry) apiv1.PoolEntry {
	return apiv1.PoolEntry{CIDR: model.CIDR, Tags: append([]string(nil), model.Tags...)}
}

func toAPIPoolPreview(model *ipmodel.IPPoolPreviewResponse) *apiv1.PoolPreviewResponse {
	if model == nil {
		return nil
	}
	items := make([]apiv1.PoolEntry, 0, len(model.Entries))
	for _, item := range model.Entries {
		items = append(items, toAPIPoolEntry(item))
	}
	return &apiv1.PoolPreviewResponse{Entries: items, NextCursor: model.NextCursor, Total: model.Total}
}

func toAPIAnalysisResult(model *ipmodel.IPAnalysisResult) *apiv1.AnalysisResult {
	if model == nil {
		return nil
	}
	items := make([]apiv1.AnalysisMatch, 0, len(model.Matches))
	for _, item := range model.Matches {
		items = append(items, apiv1.AnalysisMatch{CIDR: item.CIDR, Tags: append([]string(nil), item.Tags...)})
	}
	return &apiv1.AnalysisResult{Matched: model.Matched, Matches: items}
}

func toAPIIPInfo(model *networkcommon.IPInfoResponse) *networkcommon.IPInfoResponse {
	if model == nil {
		return nil
	}
	copy := *model
	return &copy
}

func toAPIPoolEntries(items []ipmodel.IPPoolEntry) []apiv1.PoolEntry {
	res := make([]apiv1.PoolEntry, 0, len(items))
	for _, item := range items {
		res = append(res, toAPIPoolEntry(item))
	}
	return res
}

func mapPools(res *shared.PaginationResponse[ipmodel.IPPool]) *shared.PaginationResponse[apiv1.Pool] {
	items := make([]apiv1.Pool, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIPool(item))
	}
	return &shared.PaginationResponse[apiv1.Pool]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapExports(res *shared.PaginationResponse[ipmodel.IPExport]) *shared.PaginationResponse[apiv1.Export] {
	items := make([]apiv1.Export, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIExport(item))
	}
	return &shared.PaginationResponse[apiv1.Export]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapSyncPolicies(res *shared.PaginationResponse[ipmodel.IPSyncPolicy]) *shared.PaginationResponse[apiv1.SyncPolicy] {
	items := make([]apiv1.SyncPolicy, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPISyncPolicy(item))
	}
	return &shared.PaginationResponse[apiv1.SyncPolicy]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}
