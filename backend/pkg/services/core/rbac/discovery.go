package rbac

import (
	"context"
	metav1 "homelab/pkg/apis/meta/v1"
	registryruntime "homelab/pkg/runtime/registry"
	"strings"

	discoverymodel "homelab/pkg/models/core/discovery"
)

func registerResourceDiscovery() {
	_ = registryruntime.Default().RegisterResource(registryruntime.ResourceDescriptor{
		Group: "rbac",
		Kind:  "rbac",
		Verbs: []string{"get", "list", "create", "update", "delete", "simulate", "*"},
		DiscoverFunc: func(ctx context.Context, prefix string, _ string, _ int) (*metav1.List[discoverymodel.LookupItem], error) {
			subs := []string{"serviceaccounts", "roles", "rolebindings", "simulate"}
			res := make([]discoverymodel.LookupItem, 0)
			for _, s := range subs {
				if prefix == "" || strings.HasPrefix(s, prefix) {
					res = append(res, discoverymodel.LookupItem{
						ID:   s,
						Name: s,
					})
				}
			}
			return &metav1.List[discoverymodel.LookupItem]{Items: res}, nil
		},
	})
}
