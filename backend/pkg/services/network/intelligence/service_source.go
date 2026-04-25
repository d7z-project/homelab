package intelligence

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	intelligencemodel "homelab/pkg/models/network/intelligence"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/intelligence"
	"homelab/pkg/services/network/ip"
	"path/filepath"

	"github.com/google/uuid"
)

func (s *IntelligenceService) CreateSource(ctx context.Context, source *intelligencemodel.IntelligenceSource) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/intelligence") {
		return fmt.Errorf("%w: network/intelligence", commonauth.ErrPermissionDenied)
	}
	source.ID = uuid.NewString()

	err := repo.SourceRepo.Cow(ctx, source.ID, func(res *shared.Resource[intelligencemodel.IntelligenceSourceV1Meta, intelligencemodel.IntelligenceSourceV1Status]) error {
		res.Meta = source.Meta
		res.Status.Status = shared.TaskStatusSuccess
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	})

	if err == nil {
		updated, _ := repo.SourceRepo.Get(ctx, source.ID)
		if updated != nil {
			*source = *updated
			s.updateCronJob(*source)
		}
		common.NotifyCluster(ctx, common.EventIntelligenceSourceChanged, source.ID)
	}
	return err
}

func (s *IntelligenceService) UpdateSource(ctx context.Context, source *intelligencemodel.IntelligenceSource) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/intelligence") {
		return fmt.Errorf("%w: network/intelligence", commonauth.ErrPermissionDenied)
	}

	err := repo.SourceRepo.PatchMeta(ctx, source.ID, source.Generation, func(m *intelligencemodel.IntelligenceSourceV1Meta) {
		*m = source.Meta
	})

	if err == nil {
		updated, _ := repo.SourceRepo.Get(ctx, source.ID)
		if updated != nil {
			s.updateCronJob(*updated)
		}
		common.NotifyCluster(ctx, common.EventIntelligenceSourceChanged, source.ID)
		commonaudit.FromContext(ctx).Log("UpdateIntelligence", source.Meta.Name, "Success", true)
	}
	return err
}

func (s *IntelligenceService) ScanSources(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[intelligencemodel.IntelligenceSource], error) {
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
			if status == shared.TaskStatusRunning || status == shared.TaskStatusPending {
				res.Items[i].Status.Status = status
				res.Items[i].Status.ErrorMessage = t.Error
				res.Items[i].Status.Progress = t.GetProgress()
			}
		}
	}

	return res, nil
}

func (s *IntelligenceService) DeleteSource(ctx context.Context, id string) error {
	src, _ := repo.SourceRepo.Get(ctx, id)
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
		common.NotifyCluster(ctx, common.EventMMDBUpdate, intelligencemodel.MMDBUpdatePayload{
			ID:   id,
			Type: src.Meta.Type,
		})
	}
	return nil
}
