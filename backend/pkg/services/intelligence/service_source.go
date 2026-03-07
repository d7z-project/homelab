package intelligence

import (
	"context"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/intelligence"

	"github.com/google/uuid"
)

func (s *IntelligenceService) CreateSource(ctx context.Context, source *models.IntelligenceSource) error {
	source.ID = uuid.NewString()
	source.Status = models.TaskStatusSuccess
	if err := repo.SaveSource(ctx, source); err != nil {
		return err
	}
	s.updateCronJob(*source)
	common.NotifyCluster(ctx, "intelligence_source_update", source.ID)
	return nil
}

func (s *IntelligenceService) UpdateSource(ctx context.Context, source *models.IntelligenceSource) error {
	existing, err := repo.GetSource(ctx, source.ID)
	if err != nil {
		return ErrSourceNotFound
	}
	source.Status = existing.Status
	source.LastUpdatedAt = existing.LastUpdatedAt
	source.ErrorMessage = existing.ErrorMessage

	if err := repo.SaveSource(ctx, source); err != nil {
		return err
	}

	s.updateCronJob(*source)
	common.NotifyCluster(ctx, "intelligence_source_update", source.ID)
	commonaudit.FromContext(ctx).Log("UpdateIntelligence", source.Name, "Success", true)
	return nil
}

func (s *IntelligenceService) ListSources(ctx context.Context) ([]models.IntelligenceSource, error) {
	return repo.ListSources(ctx)
}

func (s *IntelligenceService) DeleteSource(ctx context.Context, id string) error {
	if err := repo.DeleteSource(ctx, id); err != nil {
		return err
	}
	s.removeCronJob(id)
	common.NotifyCluster(ctx, "intelligence_source_delete", id)
	return nil
}
