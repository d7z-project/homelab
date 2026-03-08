package site

import (
	"context"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/site"
	"homelab/pkg/services/discovery"
	"homelab/pkg/services/rbac"
	"strings"
)

func init() {
	rbac.RegisterResourceWithVerbs("network/site", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		res := make([]models.DiscoverResult, 0)
		groups, _, err := repo.ListGroups(ctx, 1, 1000, "")
		if err != nil {
			return nil, err
		}
		for _, g := range groups {
			if strings.HasPrefix(g.ID, prefix) || strings.HasPrefix(g.Name, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: g.ID,
					Name:   g.Name,
					Final:  true,
				})
			}
		}
		return res, nil
	}, []string{"get", "list", "create", "update", "delete", "execute", "*"})

	discovery.Register("network/site/pools", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		groups, _, err := repo.ListGroups(ctx, 1, 1000, search)
		if err != nil {
			return nil, 0, err
		}
		var items []models.LookupItem
		for _, g := range groups {
			items = append(items, models.LookupItem{
				ID:          g.ID,
				Name:        g.Name,
				Description: g.Description,
			})
		}
		result, total := discovery.Paginate(items, offset, limit)
		return result, total, nil
	})
}

const (
	PoolsDir = "network/site/pools"
)

type SitePoolService struct {
	engine        *AnalysisEngine
	exportManager *ExportManager
}

func NewSitePoolService(engine *AnalysisEngine, em *ExportManager) *SitePoolService {
	return &SitePoolService{
		engine:        engine,
		exportManager: em,
	}
}
