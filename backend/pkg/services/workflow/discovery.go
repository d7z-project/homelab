package workflow

import (
	"context"
	"fmt"
	"strings"
	"sync"

	metav1 "homelab/pkg/apis/meta/v1"
	commonauth "homelab/pkg/common/auth"
	repo "homelab/pkg/repositories/workflow/actions"
	registryruntime "homelab/pkg/runtime/registry"

	discoverymodel "homelab/pkg/models/core/discovery"
	"homelab/pkg/models/shared"
)

var registerDiscoveryOnce sync.Once

func RegisterDiscovery() {
	registerDiscoveryOnce.Do(func() {
		_ = registryruntime.Default().RegisterResource(registryruntime.ResourceDescriptor{
			Group: "actions",
			Kind:  "actions",
			Verbs: []string{"get", "list", "create", "update", "delete", "execute", "*"},
			DiscoverFunc: func(ctx context.Context, prefix string, _ string, _ int) (*metav1.List[discoverymodel.LookupItem], error) {
				subs := []string{"workflows", "instances", "manifests", "probe"}
				res := make([]discoverymodel.LookupItem, 0)
				for _, s := range subs {
					if prefix == "" || strings.HasPrefix(s, prefix) {
						res = append(res, discoverymodel.LookupItem{
							ID:   s,
							Name: s,
						})
					}
				}

				for _, s := range []string{"workflows", "instances"} {
					if strings.HasPrefix(prefix, s+"/") {
						idPrefix := strings.TrimPrefix(prefix, s+"/")
						if s == "workflows" {
							workflows, err := repo.ScanAllWorkflows(ctx)
							if err == nil {
								for _, wf := range workflows {
									if idPrefix == "" || strings.HasPrefix(wf.ID, idPrefix) {
										res = append(res, discoverymodel.LookupItem{
											ID:   "workflows/" + wf.ID,
											Name: "Workflow: " + wf.Meta.Name,
										})
									}
								}
							}
						} else {
							res = append(res, discoverymodel.LookupItem{
								ID:   "instances/*",
								Name: "All Instances",
							})
						}
					}
				}

				return &metav1.List[discoverymodel.LookupItem]{Items: res}, nil
			},
		})

		_ = registryruntime.Default().RegisterLookup("actions/workflows", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
			workflows, err := repo.ScanAllWorkflows(ctx)
			if err != nil {
				return nil, err
			}
			perms := commonauth.PermissionsFromContext(ctx)
			hasGlobal := perms.IsAllowed("actions")
			var items []discoverymodel.LookupItem
			search = strings.ToLower(search)
			for _, wf := range workflows {
				if !hasGlobal && !perms.IsAllowed("actions/"+wf.ID) {
					continue
				}
				if search != "" && !strings.Contains(strings.ToLower(wf.ID), search) && !strings.Contains(strings.ToLower(wf.Meta.Name), search) {
					continue
				}
				items = append(items, discoverymodel.LookupItem{
					ID:          wf.ID,
					Name:        wf.Meta.Name,
					Description: wf.Meta.Description,
				})
			}
			return registryruntime.Paginate(items, cursor, limit), nil
		})

		registryruntime.Default().RegisterSAUsageChecker(func(ctx context.Context, id string) error {
			workflows, err := repo.ScanAllWorkflows(ctx)
			if err != nil {
				return nil
			}
			for _, wf := range workflows {
				if wf.Meta.ServiceAccountID == id {
					return fmt.Errorf("ServiceAccount '%s' is used by workflow '%s'", id, wf.Meta.Name)
				}
			}
			return nil
		})
	})
}
