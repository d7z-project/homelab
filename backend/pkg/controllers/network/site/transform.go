package site

import (
	apiv1 "homelab/pkg/apis/network/site/v1"
	networkcommon "homelab/pkg/models/network/common"
	sitemodel "homelab/pkg/models/network/site"
	"homelab/pkg/models/shared"
)

func toModelGroup(api apiv1.Group) sitemodel.SiteGroup {
	return sitemodel.SiteGroup{
		ID: api.ID,
		Meta: sitemodel.SiteGroupV1Meta{
			Name:        api.Meta.Name,
			Description: api.Meta.Description,
		},
		Status: sitemodel.SiteGroupV1Status{
			Checksum:   api.Status.Checksum,
			EntryCount: api.Status.EntryCount,
			CreatedAt:  api.Status.CreatedAt,
			UpdatedAt:  api.Status.UpdatedAt,
		},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPIGroup(model sitemodel.SiteGroup) apiv1.Group {
	return apiv1.Group{
		ID: model.ID,
		Meta: apiv1.GroupMeta{
			Name:        model.Meta.Name,
			Description: model.Meta.Description,
		},
		Status: apiv1.GroupStatus{
			Checksum:   model.Status.Checksum,
			EntryCount: model.Status.EntryCount,
			CreatedAt:  model.Status.CreatedAt,
			UpdatedAt:  model.Status.UpdatedAt,
		},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func toModelExport(api apiv1.Export) sitemodel.SiteExport {
	return sitemodel.SiteExport{
		ID: api.ID,
		Meta: sitemodel.SiteExportV1Meta{
			Name:        api.Meta.Name,
			Description: api.Meta.Description,
			Rule:        api.Meta.Rule,
			GroupIDs:    append([]string(nil), api.Meta.GroupIDs...),
		},
		Status: sitemodel.SiteExportV1Status{
			CreatedAt: api.Status.CreatedAt,
			UpdatedAt: api.Status.UpdatedAt,
		},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPIExport(model sitemodel.SiteExport) apiv1.Export {
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

func toModelSyncPolicy(api apiv1.SyncPolicy) sitemodel.SiteSyncPolicy {
	return sitemodel.SiteSyncPolicy{
		ID: api.ID,
		Meta: sitemodel.SiteSyncPolicyV1Meta{
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
		Status: sitemodel.SiteSyncPolicyV1Status{
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

func toAPISyncPolicy(model sitemodel.SiteSyncPolicy) apiv1.SyncPolicy {
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

func toModelPoolEntryRequest(api apiv1.PoolEntryRequest) sitemodel.SitePoolEntryRequest {
	return sitemodel.SitePoolEntryRequest{Type: api.Type, Value: api.Value, OldTags: append([]string(nil), api.OldTags...), NewTags: append([]string(nil), api.NewTags...)}
}

func toModelExportPreviewRequest(api apiv1.ExportPreviewRequest) sitemodel.SiteExportPreviewRequest {
	return sitemodel.SiteExportPreviewRequest{Rule: api.Rule, GroupIDs: append([]string(nil), api.GroupIDs...)}
}

func toAPIPoolEntry(model sitemodel.SitePoolEntry) apiv1.PoolEntry {
	return apiv1.PoolEntry{Type: model.Type, Value: model.Value, Tags: append([]string(nil), model.Tags...)}
}

func toAPIPoolPreview(model *sitemodel.SitePoolPreviewResponse) *apiv1.PoolPreviewResponse {
	if model == nil {
		return nil
	}
	items := make([]apiv1.PoolEntry, 0, len(model.Entries))
	for _, item := range model.Entries {
		items = append(items, toAPIPoolEntry(item))
	}
	return &apiv1.PoolPreviewResponse{Entries: items, NextCursor: model.NextCursor, Total: model.Total}
}

func toAPIAnalysisResult(model *sitemodel.SiteAnalysisResult) *apiv1.AnalysisResult {
	if model == nil {
		return nil
	}
	return &apiv1.AnalysisResult{
		Matched:  model.Matched,
		RuleType: model.RuleType,
		Pattern:  model.Pattern,
		Tags:     append([]string(nil), model.Tags...),
		DNS:      mapDNSAnalysis(model.DNS),
	}
}

func mapDNSAnalysis(model *sitemodel.SiteDNSAnalysis) *apiv1.DNSAnalysis {
	if model == nil {
		return nil
	}
	return &apiv1.DNSAnalysis{
		A:     append([]networkcommon.IPInfoResponse(nil), model.A...),
		AAAA:  append([]networkcommon.IPInfoResponse(nil), model.AAAA...),
		CNAME: append([]string(nil), model.CNAME...),
		NS:    append([]networkcommon.IPInfoResponse(nil), model.NS...),
		SOA:   append([]string(nil), model.SOA...),
	}
}

func toAPIPoolEntries(items []sitemodel.SitePoolEntry) []apiv1.PoolEntry {
	res := make([]apiv1.PoolEntry, 0, len(items))
	for _, item := range items {
		res = append(res, toAPIPoolEntry(item))
	}
	return res
}

func mapGroups(res *shared.PaginationResponse[sitemodel.SiteGroup]) *shared.PaginationResponse[apiv1.Group] {
	items := make([]apiv1.Group, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIGroup(item))
	}
	return &shared.PaginationResponse[apiv1.Group]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapExports(res *shared.PaginationResponse[sitemodel.SiteExport]) *shared.PaginationResponse[apiv1.Export] {
	items := make([]apiv1.Export, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIExport(item))
	}
	return &shared.PaginationResponse[apiv1.Export]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapSyncPolicies(res *shared.PaginationResponse[sitemodel.SiteSyncPolicy]) *shared.PaginationResponse[apiv1.SyncPolicy] {
	items := make([]apiv1.SyncPolicy, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPISyncPolicy(item))
	}
	return &shared.PaginationResponse[apiv1.SyncPolicy]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}
