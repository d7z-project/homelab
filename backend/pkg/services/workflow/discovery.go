package workflow

import (
	"context"
	"fmt"
	"strings"

	metav1 "homelab/pkg/apis/meta/v1"
	commonauth "homelab/pkg/common/auth"
	repo "homelab/pkg/repositories/workflow/actions"
	registryruntime "homelab/pkg/runtime/registry"

	discoverymodel "homelab/pkg/models/core/discovery"
	"homelab/pkg/models/shared"
)

func RegisterDiscovery(registry *registryruntime.Registry) {
	if registry == nil {
		return
	}
	_ = registry.RegisterResource(registryruntime.ResourceDescriptor{
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
						workflows, err := repo.ScanAllWorkflowsByPrefix(ctx, idPrefix)
						if err == nil {
							for _, wf := range workflows {
								res = append(res, discoverymodel.LookupItem{
									ID:   "workflows/" + wf.ID,
									Name: "Workflow: " + wf.Meta.Name,
								})
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

	_ = registry.RegisterLookup("actions/workflows", func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		workflows, err := repo.ScanWorkflows(ctx, cursor, limit, search)
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		hasGlobal := perms.IsAllowed("actions")
		var items []discoverymodel.LookupItem
		for _, wf := range workflows.Items {
			if !hasGlobal && !perms.IsAllowed("actions/"+wf.ID) {
				continue
			}
			items = append(items, discoverymodel.LookupItem{
				ID:          wf.ID,
				Name:        wf.Meta.Name,
				Description: wf.Meta.Description,
			})
		}
		return &shared.PaginationResponse[discoverymodel.LookupItem]{
			Items:      items,
			NextCursor: workflows.NextCursor,
			HasMore:    workflows.HasMore,
		}, nil
	})

	registry.RegisterSAUsageChecker(func(ctx context.Context, id string) error {
		used, workflow, err := repo.WorkflowUsesServiceAccount(ctx, id)
		if err != nil {
			return nil
		}
		if used && workflow != nil {
			return fmt.Errorf("ServiceAccount '%s' is used by workflow '%s'", id, workflow.Meta.Name)
		}
		return nil
	})
}
