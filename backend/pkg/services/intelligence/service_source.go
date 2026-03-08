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
	common.NotifyCluster(ctx, common.EventIntelligenceSourceChanged, source.ID)
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
	common.NotifyCluster(ctx, common.EventIntelligenceSourceChanged, source.ID)
	commonaudit.FromContext(ctx).Log("UpdateIntelligence", source.Name, "Success", true)
	return nil
}

func (s *IntelligenceService) ListSources(ctx context.Context) ([]models.IntelligenceSource, error) {
	sources, err := repo.ListSources(ctx)
	if err != nil {
		return nil, err
	}

	// 从内存 `manager` 获取最新运行状态、错误和进度
	for i := range sources {
		if t, ok := s.tasks.GetTask(sources[i].ID); ok {
			status := t.GetStatus()
			if status == models.TaskStatusRunning || status == models.TaskStatusPending {
				sources[i].Status = status
				sources[i].ErrorMessage = t.Error
				sources[i].Progress = t.GetProgress()
			}
		}
	}

	return sources, nil
}

func (s *IntelligenceService) DeleteSource(ctx context.Context, id string) error {
	if err := repo.DeleteSource(ctx, id); err != nil {
		return err
	}
	s.removeCronJob(id)
	common.NotifyCluster(ctx, common.EventIntelligenceSourceChanged, id)
	return nil
}
