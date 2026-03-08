package rbac

import (
	"context"
	"homelab/pkg/models"
	"homelab/pkg/services/discovery"
	"strings"
)

func init() {
	// Register rbac resources with specific verbs
	discovery.RegisterResourceWithVerbs("rbac", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		subs := []string{"serviceaccounts", "roles", "rolebindings", "simulate"}
		res := make([]models.DiscoverResult, 0)
		for _, s := range subs {
			if prefix == "" || strings.HasPrefix(s, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: s,
					Name:   s,
					Final:  true,
				})
			}
		}
		return res, nil
	}, []string{"get", "list", "create", "update", "delete", "simulate", "*"})
}
