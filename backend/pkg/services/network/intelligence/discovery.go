package intelligence

import (
	"context"

	metav1 "homelab/pkg/apis/meta/v1"
	discoverymodel "homelab/pkg/models/core/discovery"
	"homelab/pkg/models/shared"
	intrepo "homelab/pkg/repositories/network/intelligence"
	registryruntime "homelab/pkg/runtime/registry"
)

func RegisterDiscovery(registry *registryruntime.Registry) {
	if registry == nil {
		return
	}
	_ = registry.RegisterResource(registryruntime.ResourceDescriptor{
		Group:    "network",
		Resource: "intelligence",
		Kind:     "network.intelligence",
		Verbs:    []string{"list", "create", "update", "delete", "execute", "*"},
		DiscoverFunc: func(context.Context, string, string, int) (*metav1.List[discoverymodel.LookupItem], error) {
			return &metav1.List[discoverymodel.LookupItem]{Items: []discoverymodel.LookupItem{}}, nil
		},
	})

	_ = registry.RegisterLookup("network/intelligence/sources", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		resp, err := intrepo.ScanSources(ctx, cursor, limit, search)
		if err != nil {
			return nil, err
		}
		items := make([]discoverymodel.LookupItem, 0, len(resp.Items))
		for _, source := range resp.Items {
			items = append(items, discoverymodel.LookupItem{
				ID:          source.ID,
				Name:        source.Meta.Name,
				Description: source.Meta.Type,
			})
		}
		return &shared.PaginationResponse[discoverymodel.LookupItem]{
			Items:      items,
			NextCursor: resp.NextCursor,
			HasMore:    resp.HasMore,
		}, nil
	})
}
