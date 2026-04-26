package intelligence

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	taskpkg "homelab/pkg/common/task"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/intelligence"
	runtimepkg "homelab/pkg/runtime"
	"homelab/pkg/services/network/ip"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

var (
	ErrSourceNotFound = fmt.Errorf("%w: intelligence source not found", common.ErrNotFound)
)

type IntelligenceService struct {
	deps    runtimepkg.ModuleDeps
	mmdb    *ip.MMDBManager
	cron    *cron.Cron
	entries map[string]cron.EntryID
	mu      sync.Mutex
	tasks   *taskpkg.Manager[*SyncTask]
}

type SyncTask = taskpkg.SimpleTask

func NewIntelligenceService(deps runtimepkg.ModuleDeps, mmdb *ip.MMDBManager) *IntelligenceService {
	s := &IntelligenceService{
		deps:    deps,
		mmdb:    mmdb,
		cron:    cron.New(),
		entries: make(map[string]cron.EntryID),
		tasks:   taskpkg.NewManager[*SyncTask](deps, "action:intelligence_sync", "sync_tasks", "network", "intelligence"),
	}
	s.cron.Start()

	// 集群事件: 变更数据源时，刷新本节点 cron 调度 (涵盖创建、更新、删除及启停)
	common.RegisterEventHandler(common.EventIntelligenceSourceChanged, func(ctx context.Context, payload common.ResourceEventPayload) {
		src, err := repo.GetSource(ctx, payload.ID)
		if err != nil {
			s.removeCronJob(payload.ID)
			return
		}
		s.updateCronJob(*src)
	})

	return s
}

func (s *IntelligenceService) Init(ctx context.Context) error {
	sources, err := repo.ScanAllSources(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.tasks.Reconcile(ctx)
	// 同步自愈后的任务状态到资源记录
	for _, t := range s.tasks.RangeAll() {
		status := t.GetStatus()
		if status == shared.TaskStatusFailed || status == shared.TaskStatusCancelled {
			src, err := repo.GetSource(ctx, t.GetID())
			if err == nil && (src.Status.Status == shared.TaskStatusRunning || src.Status.Status == shared.TaskStatusPending) {
				src.Status.Status = status
				src.Status.ErrorMessage = t.Error
				_ = repo.SaveSource(ctx, src)
			}
		}
	}

	for i := range sources {
		src := &sources[i]
		if src.Meta.AutoUpdate && src.Meta.UpdateCron != "" {
			s.addCronJob(*src)
		}
	}

	log.Printf("IntelligenceService: initialized and scheduled tasks")
	return s.StartSyncConsumer(ctx)
}

func (s *IntelligenceService) GetTasks() *taskpkg.Manager[*SyncTask] {
	return s.tasks
}

func (s *IntelligenceService) CancelTask(id string) bool {
	return s.tasks.CancelTask(id)
}
