package ip

import (
	"context"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	rbacmodel "homelab/pkg/models/core/rbac"
	ipmodel "homelab/pkg/models/network/ip"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/ip"

	"sync"
	"time"

	"homelab/pkg/common/task"

	"github.com/robfig/cron/v3"
)

const (
	PoolsDir       = "network/ip/pools"
	MaxPoolEntries = 2000000
)

type IPPoolService struct {
	cron           *cron.Cron
	cronIDs        map[string]cron.EntryID
	cronLock       sync.Mutex
	exportManager  *ExportManager
	analysisEngine *AnalysisEngine
	syncTasks      *task.Manager[*SyncTask]
}

type SyncTask struct {
	ID        string            `json:"id"`
	Status    shared.TaskStatus `json:"status"`
	Progress  float64           `json:"progress"`
	Error     string            `json:"error"`
	CreatedAt time.Time         `json:"createdAt"`
	mu        sync.Mutex
}

func (t *SyncTask) GetID() string { return t.ID }
func (t *SyncTask) GetStatus() shared.TaskStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Status
}
func (t *SyncTask) SetStatus(status shared.TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status
}
func (t *SyncTask) SetError(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Error = msg
}
func (t *SyncTask) GetCreatedAt() time.Time { return t.CreatedAt }
func (t *SyncTask) GetProgress() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Progress
}
func (t *SyncTask) SetProgress(progress float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Progress = progress
}

var _ shared.TaskInfo = (*SyncTask)(nil)

func NewIPPoolService(ae *AnalysisEngine, em *ExportManager) *IPPoolService {
	svc := &IPPoolService{
		analysisEngine: ae,
		exportManager:  em,
		cron:           cron.New(),
		cronIDs:        make(map[string]cron.EntryID),
		syncTasks:      task.NewManager[*SyncTask]("action:ip_sync", "sync_tasks", "network", "ip"),
	}

	// 集群事件: 变更同步策略时，刷新本节点 cron 调度 (涵盖创建、更新、删除及启停)
	common.RegisterEventHandler(common.EventIPSyncPolicyChanged, func(ctx context.Context, policyID string) {
		policy, err := repo.SyncPolicyRepo.Get(ctx, policyID)
		if err != nil {
			svc.removeCronJob(policyID)
			return
		}
		if policy.Meta.Enabled {
			svc.addCronJob(*policy)
		} else {
			svc.removeCronJob(policyID)
		}
	})

	// 集群事件: 异步触发同步
	common.RegisterEventHandler(common.EventIPSyncRun, func(ctx context.Context, policyID string) {
		// 注入系统权限
		sysCtx := commonauth.WithAuth(ctx, &commonauth.AuthContext{
			Type: "sa",
			ID:   "system",
		})
		sysCtx = commonauth.WithPermissions(sysCtx, &rbacmodel.ResourcePermissions{AllowedAll: true})

		// 由于我们迁移到了 task.Manager，它天然具备防重和分布式同步能力
		go svc.doSync(sysCtx, policyID)
	})

	return svc
}

func (s *IPPoolService) lockPool(ctx context.Context, id string) (func(), error) {
	lockKey := "network:ip:pool:" + id
	for {
		release := common.Locker.TryLock(ctx, lockKey)
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

func (s *IPPoolService) GetSyncTasks() *task.Manager[*SyncTask] {
	return s.syncTasks
}

func (s *IPPoolService) StartSyncRunner(ctx context.Context) {
	s.cron.Start()
	// 加载所有启用的策略
	// 注入一个系统权限的 context
	sysCtx := commonauth.WithAuth(ctx, &commonauth.AuthContext{
		Type: "sa",
		ID:   "system",
	})
	sysCtx = commonauth.WithPermissions(sysCtx, &rbacmodel.ResourcePermissions{AllowedAll: true})

	// 1. 状态自愈 (Reconciliation) 使用全新框架接管
	s.syncTasks.Reconcile(sysCtx)
	for _, t := range s.syncTasks.RangeAll() {
		status := t.GetStatus()
		if status == shared.TaskStatusFailed || status == shared.TaskStatusCancelled {
			p, err := repo.SyncPolicyRepo.Get(sysCtx, t.GetID())
			if err == nil && (p.Status.LastStatus == shared.TaskStatusRunning || p.Status.LastStatus == shared.TaskStatusPending) {
				p.Status.LastStatus = status
				p.Status.ErrorMessage = t.Error
				p.Status.LastRunAt = time.Now()
				_ = repo.SyncPolicyRepo.Cow(sysCtx, p.ID, func(res *ipmodel.IPSyncPolicy) error { res.Meta = p.Meta; res.Status = p.Status; return nil })
			}
		}
	}

	// 然后调度启用的策略
	policiesRes, err := repo.SyncPolicyRepo.List(sysCtx, "", 10000, nil)
	if err == nil {
		for _, p := range policiesRes.Items {
			if p.Meta.Enabled {
				s.addCronJob(p)
			}
		}
	}
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
		// 注入一个系统权限的 context
		ctx = commonauth.WithAuth(ctx, &commonauth.AuthContext{
			Type: "sa",
			ID:   "system",
		})
		ctx = commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{AllowedAll: true})
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
