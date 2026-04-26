package site

import (
	"context"
	commonauth "homelab/pkg/common/auth"
	taskpkg "homelab/pkg/common/task"
	sitemodel "homelab/pkg/models/network/site"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/site"
	runtimepkg "homelab/pkg/runtime"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"homelab/pkg/common"
)

const (
	PoolsDir = "network/site/pools"
)

type SitePoolService struct {
	deps          runtimepkg.ModuleDeps
	engine        *AnalysisEngine
	exportManager *ExportManager
	syncTasks     *taskpkg.Manager[*SyncTask]

	cron     *cron.Cron
	cronIDs  map[string]cron.EntryID
	cronLock sync.Mutex
}

type SyncTask = taskpkg.SimpleTask

func NewSitePoolService(deps runtimepkg.ModuleDeps, engine *AnalysisEngine, em *ExportManager) *SitePoolService {
	s := &SitePoolService{
		deps:          deps,
		engine:        engine,
		exportManager: em,
		syncTasks:     taskpkg.NewManager[*SyncTask](deps, "action:site_sync", "sync_tasks", "network", "site"),
		cron:          cron.New(),
		cronIDs:       make(map[string]cron.EntryID),
	}
	s.cron.Start()

	// 集群事件: 变更同步策略时，刷新本节点 cron 调度 (涵盖创建、更新、删除及启停)
	common.RegisterEventHandler(common.EventSiteSyncPolicyChanged, func(ctx context.Context, payload common.ResourceEventPayload) {
		policy, err := repo.GetSyncPolicy(ctx, payload.ID)
		if err != nil {
			s.removeCronJob(payload.ID)
			return
		}
		if policy.Meta.Enabled {
			s.addCronJob(*policy)
		} else {
			s.removeCronJob(payload.ID)
		}
	})

	return s
}

func (s *SitePoolService) GetSyncTaskManager() *taskpkg.Manager[*SyncTask] {
	return s.syncTasks
}

func (s *SitePoolService) Start(ctx context.Context) error {
	sysCtx := commonauth.WithSystemSA(ctx)

	s.syncTasks.Reconcile(sysCtx)
	for _, t := range s.syncTasks.RangeAll() {
		status := t.GetStatus()
		if status == shared.TaskStatusFailed || status == shared.TaskStatusCancelled {
			p, err := repo.GetSyncPolicy(sysCtx, t.GetID())
			if err == nil && (p.Status.LastStatus == shared.TaskStatusRunning || p.Status.LastStatus == shared.TaskStatusPending) {
				p.Status.LastStatus = status
				p.Status.ErrorMessage = t.Error
				p.Status.LastRunAt = time.Now()
				_ = repo.SaveSyncPolicy(sysCtx, p)
			}
		}
	}

	policies, _ := repo.ScanSyncPolicies(sysCtx, "", 1000, "")
	if policies != nil {
		for _, p := range policies.Items {
			if p.Meta.Enabled {
				s.addCronJob(p)
			}
		}
	}
	return s.StartSyncConsumer(ctx)
}

func (s *SitePoolService) addCronJob(p sitemodel.SiteSyncPolicy) {
	s.cronLock.Lock()
	defer s.cronLock.Unlock()

	if id, ok := s.cronIDs[p.ID]; ok {
		s.cron.Remove(id)
	}

	lockKey := "site_sync_" + p.ID
	id, err := common.AddDistributedCronJob(s.cron, p.Meta.Cron, lockKey, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		ctx = commonauth.WithSystemSA(ctx)
		_ = s.Sync(ctx, p.ID)
	})
	if err == nil {
		s.cronIDs[p.ID] = id
	}
}

func (s *SitePoolService) removeCronJob(id string) {
	s.cronLock.Lock()
	defer s.cronLock.Unlock()
	if entryID, ok := s.cronIDs[id]; ok {
		s.cron.Remove(entryID)
		delete(s.cronIDs, id)
	}
}
