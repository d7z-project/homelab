package ip

import (
	"context"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	taskpkg "homelab/pkg/common/task"
	ipmodel "homelab/pkg/models/network/ip"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/ip"
	runtimepkg "homelab/pkg/runtime"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

const (
	PoolsDir       = "network/ip/pools"
	MaxPoolEntries = 2000000
)

type IPPoolService struct {
	deps           runtimepkg.ModuleDeps
	cron           *cron.Cron
	cronIDs        map[string]cron.EntryID
	cronLock       sync.Mutex
	exportManager  *ExportManager
	analysisEngine *AnalysisEngine
	syncTasks      *taskpkg.Manager[*SyncTask]
}

type SyncTask = taskpkg.SimpleTask

func NewIPPoolService(deps runtimepkg.ModuleDeps, ae *AnalysisEngine, em *ExportManager) *IPPoolService {
	svc := &IPPoolService{
		deps:           deps,
		analysisEngine: ae,
		exportManager:  em,
		cron:           cron.New(),
		cronIDs:        make(map[string]cron.EntryID),
		syncTasks:      taskpkg.NewManager[*SyncTask](deps, "action:ip_sync", "sync_tasks", "network", "ip"),
	}

	// 集群事件: 变更同步策略时，刷新本节点 cron 调度 (涵盖创建、更新、删除及启停)
	common.RegisterEventHandler(common.EventIPSyncPolicyChanged, func(ctx context.Context, payload common.ResourceEventPayload) {
		policy, err := repo.GetSyncPolicy(ctx, payload.ID)
		if err != nil {
			svc.removeCronJob(payload.ID)
			return
		}
		if policy.Meta.Enabled {
			svc.addCronJob(*policy)
		} else {
			svc.removeCronJob(payload.ID)
		}
	})

	return svc
}

func (s *IPPoolService) lockPool(ctx context.Context, id string) (func(), error) {
	lockKey := "network:ip:pool:" + id
	for {
		release := s.deps.Locker.TryLock(ctx, lockKey)
		if release != nil {
			return release, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (s *IPPoolService) GetSyncTasks() *taskpkg.Manager[*SyncTask] {
	return s.syncTasks
}

func (s *IPPoolService) Start(ctx context.Context) error {
	s.cron.Start()
	sysCtx := commonauth.WithSystemSA(ctx)

	// 启动前先把遗留的 pending/running 任务收敛回最终状态。
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

	// 再恢复启用策略的 cron 调度。
	policies, err := repo.ScanAllSyncPolicies(sysCtx)
	if err == nil {
		for _, p := range policies {
			if p.Meta.Enabled {
				s.addCronJob(p)
			}
		}
	}
	return s.StartSyncConsumer(ctx)
}

func (s *IPPoolService) addCronJob(p ipmodel.IPSyncPolicy) {
	s.cronLock.Lock()
	defer s.cronLock.Unlock()

	// 如果已存在，先删除
	if id, ok := s.cronIDs[p.ID]; ok {
		s.cron.Remove(id)
	}

	lockKey := "ip_sync_" + p.ID
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

func (s *IPPoolService) removeCronJob(id string) {
	s.cronLock.Lock()
	defer s.cronLock.Unlock()
	if entryID, ok := s.cronIDs[id]; ok {
		s.cron.Remove(entryID)
		delete(s.cronIDs, id)
	}
}
