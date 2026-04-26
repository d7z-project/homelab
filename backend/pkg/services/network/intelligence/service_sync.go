package intelligence

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	taskpkg "homelab/pkg/common/task"
	intelligencemodel "homelab/pkg/models/network/intelligence"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/intelligence"
	"homelab/pkg/services/network/ip"
	"io"
	"net/http"
	"path/filepath"
	"time"
)

func (s *IntelligenceService) SyncSource(ctx context.Context, id string) error {
	source, err := repo.GetSource(ctx, id)
	if err != nil {
		return err
	}

	// 使用框架校验并处理并发
	existingTask, ok := s.tasks.GetTask(id)
	if ok {
		status := existingTask.GetStatus()
		if status == shared.TaskStatusPending || status == shared.TaskStatusRunning {
			lockKey := "action:intelligence_sync:" + id
			if release := s.deps.Locker.TryLock(ctx, lockKey); release != nil {
				// 僵尸任务
				s.tasks.CancelTask(id)
				release()
			} else {
				return fmt.Errorf("sync is already in progress for source: %s", source.Meta.Name)
			}
		}
	}

	task := &SyncTask{
		ID:        id,
		Status:    shared.TaskStatusPending,
		CreatedAt: time.Now(),
	}
	s.tasks.AddTask(task)

	source.Status.Status = shared.TaskStatusPending
	source.Status.Progress = 0
	source.Status.ErrorMessage = ""
	source.Status.QueueTopic = intelligenceSyncTopic
	source.Status.QueueMessageID = ""
	source.Status.QueuedAt = nil
	source.Status.DispatchedAt = nil
	_ = repo.SaveSource(ctx, source)

	payload, err := json.Marshal(syncJob{SourceID: id})
	if err != nil {
		task.SetStatus(shared.TaskStatusFailed)
		task.SetError(err.Error())
		s.tasks.Save()
		source.Status.Status = shared.TaskStatusFailed
		source.Status.ErrorMessage = err.Error()
		_ = repo.SaveSource(ctx, source)
		return fmt.Errorf("failed to encode intelligence sync job: %w", err)
	}
	messageID, err := s.deps.Queue.Publish(ctx, intelligenceSyncTopic, string(payload), nil)
	if err != nil {
		task.SetStatus(shared.TaskStatusFailed)
		task.SetError(err.Error())
		s.tasks.Save()
		source.Status.Status = shared.TaskStatusFailed
		source.Status.ErrorMessage = err.Error()
		_ = repo.SaveSource(ctx, source)
		return fmt.Errorf("failed to enqueue intelligence sync job: %w", err)
	}
	queuedAt := time.Now()
	source.Status.QueueMessageID = messageID
	source.Status.QueuedAt = &queuedAt
	_ = repo.SaveSource(ctx, source)

	commonaudit.FromContext(ctx).Log("SyncIntelligence", source.Meta.Name, "Queued", true)
	return nil
}

func (s *IntelligenceService) runDownload(bgCtx context.Context, id string) {
	s.tasks.RunTask(bgCtx, id, func(taskCtx context.Context, task *SyncTask) error {
		source, err := repo.GetSource(taskCtx, id)
		var finalErr error
		defer func() {
			// 这里必须使用 Background 因为 taskCtx 已经被 Cancel 了，如果用 taskCtx 会导致 DB 操作失败
			source, _ := repo.GetSource(context.Background(), id)
			if source == nil {
				return
			}
			if errors.Is(taskCtx.Err(), context.Canceled) {
				source.Status.Status = shared.TaskStatusCancelled
				source.Status.ErrorMessage = "Task cancelled manually"
			} else if finalErr != nil {
				source.Status.Status = shared.TaskStatusFailed
				source.Status.ErrorMessage = finalErr.Error()
			} else {
				source.Status.Status = shared.TaskStatusSuccess
				source.Status.ErrorMessage = ""
				source.Status.LastUpdatedAt = time.Now()
			}
			_ = repo.SaveSource(context.Background(), source)
		}()

		if err != nil || source == nil {
			finalErr = fmt.Errorf("source not found")
			return finalErr
		}
		source.Status.Status = shared.TaskStatusRunning
		source.Status.ErrorMessage = ""
		_ = repo.SaveSource(taskCtx, source)

		// SSRF 校验
		if err := common.ValidateSourceURL(source.Meta.URL, source.Meta.Config); err != nil {
			finalErr = err
			return finalErr
		}

		finalErr = s.downloadFile(taskCtx, source, task)

		if finalErr == nil {
			common.UpdateGlobalVersion(taskCtx, "network/intelligence/mmdb")
			// 注意：此处必须使用 context.Background()，而非 taskCtx。
			// 因为 taskCtx 在任务完成后会被 cancel，使用已取消的 ctx 发布事件
			// 会导致 MemorySubscriber.Publish 的 select 直接走 ctx.Done() 分支，
			// 消息被静默丢弃，MMDB Reload 永远无法被触发。
			common.NotifyCluster(context.Background(), common.EventMMDBUpdate, intelligencemodel.MMDBUpdatePayload{
				ID:   source.ID,
				Type: source.Meta.Type,
			})
		}
		return finalErr
	})
}

func (s *IntelligenceService) downloadFile(ctx context.Context, source *intelligencemodel.IntelligenceSource, task *SyncTask) error {
	req, err := http.NewRequestWithContext(ctx, "GET", source.Meta.URL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 300 * time.Second, Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
	}}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http status %d", resp.StatusCode)
	}

	targetPath := filepath.Join(ip.MMDBDir, source.ID+".mmdb")
	_ = s.deps.FS.MkdirAll(ip.MMDBDir, 0755)

	// 这里也可以做成先写临时文件，成功后再重命名，避免写一半被 Cancel 导致源文件损坏
	tempPath := targetPath + ".tmp"
	f, err := s.deps.FS.Create(tempPath)
	if err != nil {
		return err
	}

	reader := taskpkg.NewProgressReader[*SyncTask](resp.Body, resp.ContentLength, task, s.tasks)

	_, err = io.Copy(f, reader)
	f.Close()

	if err != nil {
		_ = s.deps.FS.Remove(tempPath)
		return err
	}

	return s.deps.FS.Rename(tempPath, targetPath)
}
