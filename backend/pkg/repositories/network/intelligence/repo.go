package intelligence

import (
	"context"
	"homelab/pkg/common"
	"strings"

	intelligencemodel "homelab/pkg/models/network/intelligence"
	"homelab/pkg/models/shared"

	"gopkg.d7z.net/middleware/kv"
)

var sourceRepo *common.ResourceRepository[intelligencemodel.IntelligenceSourceV1Meta, intelligencemodel.IntelligenceSourceV1Status]

func Configure(db kv.KV) {
	sourceRepo = common.NewResourceRepository[intelligencemodel.IntelligenceSourceV1Meta, intelligencemodel.IntelligenceSourceV1Status](db, "network", "IntelligenceSource")
}

func GetSource(ctx context.Context, id string) (*intelligencemodel.IntelligenceSource, error) {
	return sourceRepo.Get(ctx, id)
}

func SaveSource(ctx context.Context, source *intelligencemodel.IntelligenceSource) error {
	return sourceRepo.Save(ctx, source)
}

func ScanAllSources(ctx context.Context) ([]intelligencemodel.IntelligenceSource, error) {
	return sourceRepo.ListAll(ctx)
}

func ScanSources(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[intelligencemodel.IntelligenceSource], error) {
	search = strings.ToLower(search)
	return sourceRepo.List(ctx, cursor, limit, func(s *intelligencemodel.IntelligenceSource) bool {
		return search == "" || strings.Contains(strings.ToLower(s.Meta.Name), search) || strings.Contains(strings.ToLower(s.ID), search)
	})
}

func DeleteSource(ctx context.Context, id string) error {
	return sourceRepo.Delete(ctx, id)
}
