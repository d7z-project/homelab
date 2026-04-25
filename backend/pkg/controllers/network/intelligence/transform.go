package intelligence

import (
	apiv1 "homelab/pkg/apis/network/intelligence/v1"
	intelligencemodel "homelab/pkg/models/network/intelligence"
	"homelab/pkg/models/shared"
)

func toModelSource(api apiv1.Source) intelligencemodel.IntelligenceSource {
	return intelligencemodel.IntelligenceSource{
		ID: api.ID,
		Meta: intelligencemodel.IntelligenceSourceV1Meta{
			Name:       api.Meta.Name,
			Type:       api.Meta.Type,
			URL:        api.Meta.URL,
			Enabled:    api.Meta.Enabled,
			AutoUpdate: api.Meta.AutoUpdate,
			UpdateCron: api.Meta.UpdateCron,
			Config:     api.Meta.Config,
		},
		Status: intelligencemodel.IntelligenceSourceV1Status{
			LastUpdatedAt: api.Status.LastUpdatedAt,
			Status:        api.Status.Status,
			Progress:      api.Status.Progress,
			ErrorMessage:  api.Status.ErrorMessage,
		},
		Generation:      api.Generation,
		ResourceVersion: api.ResourceVersion,
	}
}

func toAPISource(model intelligencemodel.IntelligenceSource) apiv1.Source {
	return apiv1.Source{
		ID: model.ID,
		Meta: apiv1.SourceMeta{
			Name:       model.Meta.Name,
			Type:       model.Meta.Type,
			URL:        model.Meta.URL,
			Enabled:    model.Meta.Enabled,
			AutoUpdate: model.Meta.AutoUpdate,
			UpdateCron: model.Meta.UpdateCron,
			Config:     model.Meta.Config,
		},
		Status: apiv1.SourceStatus{
			LastUpdatedAt: model.Status.LastUpdatedAt,
			Status:        model.Status.Status,
			Progress:      model.Status.Progress,
			ErrorMessage:  model.Status.ErrorMessage,
		},
		Generation:      model.Generation,
		ResourceVersion: model.ResourceVersion,
	}
}

func mapSources(res *shared.PaginationResponse[intelligencemodel.IntelligenceSource]) *shared.PaginationResponse[apiv1.Source] {
	items := make([]apiv1.Source, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPISource(item))
	}
	return &shared.PaginationResponse[apiv1.Source]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}
