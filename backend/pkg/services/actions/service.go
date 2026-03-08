package actions

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/actions"
	"homelab/pkg/services/discovery"
	"strings"
)

func init() {
	discovery.RegisterResourceWithVerbs("actions", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {

		// prefix is everything after "actions/"
		subs := []string{"workflows", "instances", "manifests", "probe"}
		res := make([]models.DiscoverResult, 0)
		for _, s := range subs {
			if prefix == "" || strings.HasPrefix(s, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: s,
					Name:   s,
					Final:  false, // can have instances like workflows/ID
				})
			}
		}

		// If prefix starts with a sub-resource, suggest IDs
		for _, s := range []string{"workflows", "instances"} {
			if strings.HasPrefix(prefix, s+"/") {
				idPrefix := strings.TrimPrefix(prefix, s+"/")
				if s == "workflows" {
					workflows, err := repo.ScanAllWorkflows(ctx)
					if err == nil {
						for _, wf := range workflows {
							if idPrefix == "" || strings.HasPrefix(wf.ID, idPrefix) {
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

	discovery.Register("actions/workflows", func(ctx context.Context, search string, cursor string, limit int) (*models.PaginationResponse[models.LookupItem], error) {
		workflows, err := repo.ScanAllWorkflows(ctx)
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		hasGlobal := perms.IsAllowed("actions")
		var items []models.LookupItem
		search = strings.ToLower(search)
		for _, wf := range workflows {
			if !hasGlobal && !perms.IsAllowed("actions/"+wf.ID) {
				continue
			}
			if search != "" && !strings.Contains(strings.ToLower(wf.ID), search) && !strings.Contains(strings.ToLower(wf.Name), search) {
				continue
			}
			items = append(items, models.LookupItem{
				ID:          wf.ID,
				Name:        wf.Name,
				Description: wf.Description,
			})
		}
		return discovery.Paginate(items, cursor, limit), nil
	})

	discovery.RegisterSAUsageChecker(func(ctx context.Context, id string) error {
		workflows, err := repo.ScanAllWorkflows(ctx)
		if err != nil {
			return nil
		}
		for _, wf := range workflows {
			if wf.ServiceAccountID == id {
				return fmt.Errorf("ServiceAccount '%s' is used by workflow '%s'", id, wf.Name)
			}
		}
		return nil
	})
}
