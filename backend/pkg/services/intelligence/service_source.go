package intelligence

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/intelligence"
	"homelab/pkg/services/ip"
	"path/filepath"

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

func (s *IntelligenceService) ScanSources(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.IntelligenceSource], error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/intelligence") {
		return nil, fmt.Errorf("%w: network/intelligence", commonauth.ErrPermissionDenied)
	}
	res, err := repo.ScanSources(ctx, cursor, limit, search)
	if err != nil {
		return nil, err
	}

	// 从内存 `manager` 获取最新运行状态、错误和进度
	for i := range res.Items {
		if t, ok := s.tasks.GetTask(res.Items[i].ID); ok {
			status := t.GetStatus()
			if status == models.TaskStatusRunning || status == models.TaskStatusPending {
				res.Items[i].Status = status
				res.Items[i].ErrorMessage = t.Error
				res.Items[i].Progress = t.GetProgress()
			}
		}
	}

	return res, nil
}

func (s *IntelligenceService) DeleteSource(ctx context.Context, id string) error {
	src, _ := repo.GetSource(ctx, id)
	if err := repo.DeleteSource(ctx, id); err != nil {
		return err
	}
	s.removeCronJob(id)

	// 物理删除文件
	path := filepath.Join(ip.MMDBDir, id+".mmdb")
	_ = common.FS.Remove(path)

	// 通知集群：配置已变且库需卸载
	common.NotifyCluster(ctx, common.EventIntelligenceSourceChanged, id)
	if src != nil {
		common.NotifyCluster(ctx, common.EventMMDBUpdate, models.MMDBUpdatePayload{
			ID:   id,
			Type: src.Type,
		})
	}
	return nil
}
