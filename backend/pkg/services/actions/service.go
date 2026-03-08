package actions

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/actions"
	"homelab/pkg/services/discovery"
	"homelab/pkg/services/rbac"
	"strings"
)

func init() {
	rbac.RegisterResourceWithVerbs("actions", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {

		// prefix is everything after "actions/"
		subs := []string{"workflows", "instances", "manifests", "probe"}
		res := make([]models.DiscoverResult, 0)
		for _, s := range subs {
			if strings.HasPrefix(s, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: s,
					Name:   s,
					Final:  true,
				})
			}
		}

		// If prefix starts with a sub-resource, suggest IDs
		for _, s := range []string{"workflows", "instances"} {
			if strings.HasPrefix(prefix, s+"/") {
				idPrefix := strings.TrimPrefix(prefix, s+"/")
				if s == "workflows" {
					workflows, err := repo.ListWorkflows(ctx)
					if err == nil {
						for _, wf := range workflows {
							if strings.HasPrefix(wf.ID, idPrefix) {
								res = append(res, models.DiscoverResult{
									FullID: "workflows/" + wf.ID,
									Name:   "Workflow: " + wf.Name,
									Final:  true,
								})
							}
						}
					}
				} else {
					res = append(res, models.DiscoverResult{
						FullID: "instances/*",
						Name:   "All Instances",
						Final:  true,
					})
				}
			}
		}

		return res, nil
	}, []string{"get", "list", "create", "update", "delete", "execute", "*"})

	discovery.Register("actions/workflows", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions") {
			return nil, 0, fmt.Errorf("%w: actions", commonauth.ErrPermissionDenied)
		}
		workflows, err := repo.ListWorkflows(ctx)
		if err != nil {
			return nil, 0, err
		}
		var items []models.LookupItem
		search = strings.ToLower(search)
		for _, wf := range workflows {
			if search != "" && !strings.Contains(strings.ToLower(wf.ID), search) && !strings.Contains(strings.ToLower(wf.Name), search) {
				continue
			}
			items = append(items, models.LookupItem{
				ID:          wf.ID,
				Name:        wf.Name,
				Description: wf.Description,
			})
		}
		result, total := discovery.Paginate(items, offset, limit)
		return result, total, nil
	})
}
