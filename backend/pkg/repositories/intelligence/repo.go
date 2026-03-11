package intelligence

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"
)

var SourceRepo = common.NewBaseRepository[models.IntelligenceSourceV1Meta, models.IntelligenceSourceV1Status]("network", "IntelligenceSource")

func GetSource(ctx context.Context, id string) (*models.IntelligenceSource, error) {
	return SourceRepo.Get(ctx, id)
}

func SaveSource(ctx context.Context, source *models.IntelligenceSource) error {
	return SourceRepo.Cow(ctx, source.ID, func(res *models.IntelligenceSource) error {
		res.Meta = source.Meta
		res.Status = source.Status
		return nil
	})
}

func ScanAllSources(ctx context.Context) ([]models.IntelligenceSource, error) {
	res, err := SourceRepo.List(ctx, "", 10000, nil)
	if err != nil {
		return nil, err
	}
	return res.Items, nil
}

func ScanSources(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.IntelligenceSource], error) {
	search = strings.ToLower(search)
	return SourceRepo.List(ctx, cursor, limit, func(s *models.IntelligenceSource) bool {
		return search == "" || strings.Contains(strings.ToLower(s.Meta.Name), search) || strings.Contains(strings.ToLower(s.ID), search)
	})
}

func DeleteSource(ctx context.Context, id string) error {
	return SourceRepo.Delete(ctx, id)
}
