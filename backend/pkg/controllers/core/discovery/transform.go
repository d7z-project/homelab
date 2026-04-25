package discovery

import (
	apiv1 "homelab/pkg/apis/core/discovery/v1"
	discoverymodel "homelab/pkg/models/core/discovery"
	"homelab/pkg/models/shared"
)

func toModelLookupRequest(api apiv1.LookupRequest) discoverymodel.LookupRequest {
	return discoverymodel.LookupRequest{
		Code:   api.Code,
		Search: api.Search,
		Cursor: api.Cursor,
		Limit:  api.Limit,
	}
}

func mapLookupItems(res *shared.PaginationResponse[discoverymodel.LookupItem]) *shared.PaginationResponse[apiv1.LookupItem] {
	items := make([]apiv1.LookupItem, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, apiv1.LookupItem{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Icon:        item.Icon,
		})
	}
	return &shared.PaginationResponse[apiv1.LookupItem]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}

func mapDiscoverResults(items []discoverymodel.DiscoverResult) []apiv1.DiscoverResult {
	res := make([]apiv1.DiscoverResult, 0, len(items))
	for _, item := range items {
		res = append(res, apiv1.DiscoverResult{
			FullID: item.FullID,
			Name:   item.Name,
			Final:  item.Final,
		})
	}
	return res
}
