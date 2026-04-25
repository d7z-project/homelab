package audit

import (
	"context"
	"strings"
	"sync"

	metav1 "homelab/pkg/apis/meta/v1"
	registryruntime "homelab/pkg/runtime/registry"

	discoverymodel "homelab/pkg/models/core/discovery"
)

var registerDiscoveryOnce sync.Once

func RegisterDiscovery() {
	registerDiscoveryOnce.Do(func() {
		_ = registryruntime.Default().RegisterResource(registryruntime.ResourceDescriptor{
			Group: "audit",
			Kind:  "audit",
			Verbs: []string{"get", "list", "delete", "*"},
			DiscoverFunc: func(ctx context.Context, prefix string, _ string, _ int) (*metav1.List[discoverymodel.LookupItem], error) {
				subs := []string{"logs"}
				res := make([]discoverymodel.LookupItem, 0)
				for _, s := range subs {
					if strings.HasPrefix(s, prefix) {
						res = append(res, discoverymodel.LookupItem{
							ID:   s,
							Name: s,
						})
					}
				}
				return &metav1.List[discoverymodel.LookupItem]{Items: res}, nil
			},
		})
	})
}
