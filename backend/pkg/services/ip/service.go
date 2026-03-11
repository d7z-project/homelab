package ip

import (
	"context"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
	"homelab/pkg/services/discovery"
	"strings"

	"sync"
	"time"

	"homelab/pkg/common/task"

	"github.com/robfig/cron/v3"
)

func init() {
	discovery.RegisterResourceWithVerbs("network/ip", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		res := make([]models.DiscoverResult, 0)
		groupsRes, err := repo.PoolRepo.List(ctx, "", 1000, nil)
		if err != nil {
			return nil, err
		}
		for _, g := range groupsRes.Items {
			if prefix == "" || strings.HasPrefix(g.ID, prefix) || strings.HasPrefix(g.Meta.Name, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: g.ID,
					Name:   g.Meta.Name,
					Final:  true,
				})
			}
		}
		return res, nil
	}, []string{"get", "list", "create", "update", "delete", "execute", "*"})

	discovery.Register("network/ip/pools", func(ctx context.Context, search string, cursor string, limit int) (*models.PaginationResponse[models.LookupItem], error) {
		groupsRes, err := repo.PoolRepo.List(ctx, "", 1000, nil)
		// The original scan grouped filtered by search string, we would need to filter here.
		// For now we pass nil to filter parameter and filter manually below although it's incomplete without the filter implementation.
		// Wait actually, List accepts a filter func
		groupsRes, err = repo.PoolRepo.List(ctx, "", 1000, func(p *models.IPPool) bool {
			return search == "" || strings.Contains(strings.ToLower(p.Meta.Name), strings.ToLower(search)) || strings.Contains(strings.ToLower(p.ID), strings.ToLower(search))
		})
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		hasGlobal := perms.IsAllowed("network/ip")
		var items []models.LookupItem
		for _, g := range groupsRes.Items {
			if hasGlobal || perms.IsAllowed("network/ip/"+g.ID) {
				items = append(items, models.LookupItem{
					ID:          g.ID,
					Name:        g.Meta.Name,
					Description: g.Meta.Description,
				})
			}
		}
		return discovery.Paginate(items, cursor, limit), nil
	})
}

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
	Status    models.TaskStatus `json:"status"`
	Progress  float64           `json:"progress"`
	Error     string            `json:"error"`
	CreatedAt time.Time         `json:"createdAt"`
	mu        sync.Mutex
}

func (t *SyncTask) GetID() string { return t.ID }
func (t *SyncTask) GetStatus() models.TaskStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Status
}
func (t *SyncTask) SetStatus(status models.TaskStatus) {
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

var _ models.TaskInfo = (*SyncTask)(nil)

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
		sysCtx = commonauth.WithPermissions(sysCtx, &models.ResourcePermissions{AllowedAll: true})

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
	sysCtx = commonauth.WithPermissions(sysCtx, &models.ResourcePermissions{AllowedAll: true})

	// 1. 状态自愈 (Reconciliation) 使用全新框架接管
	s.syncTasks.Reconcile(sysCtx)
	for _, t := range s.syncTasks.RangeAll() {
		status := t.GetStatus()
		if status == models.TaskStatusFailed || status == models.TaskStatusCancelled {
			p, err := repo.SyncPolicyRepo.Get(sysCtx, t.GetID())
			if err == nil && (p.Status.LastStatus == models.TaskStatusRunning || p.Status.LastStatus == models.TaskStatusPending) {
				p.Status.LastStatus = status
				p.Status.ErrorMessage = t.Error
				p.Status.LastRunAt = time.Now()
				_ = repo.SyncPolicyRepo.Cow(sysCtx, p.ID, func(res *models.IPSyncPolicy) error { res.Meta = p.Meta; res.Status = p.Status; return nil })
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

func (s *IPPoolService) addCronJob(p models.IPSyncPolicy) {
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
		ctx = commonauth.WithPermissions(ctx, &models.ResourcePermissions{AllowedAll: true})
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
