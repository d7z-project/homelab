package site

import (
	"context"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/site"
	"homelab/pkg/services/discovery"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"homelab/pkg/common"
	taskpkg "homelab/pkg/common/task"
)

func init() {
	discovery.RegisterResourceWithVerbs("network/site", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		res := make([]models.DiscoverResult, 0)
		resp, err := repo.ScanGroups(ctx, "", 1000, "")
		if err != nil {
			return nil, err
		}
		for _, g := range resp.Items {
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

	discovery.Register("network/site/pools", func(ctx context.Context, search string, cursor string, limit int) (*models.PaginationResponse[models.LookupItem], error) {
		resp, err := repo.ScanGroups(ctx, "", 1000, search)
		if err != nil {
			return nil, err
		}
		perms := commonauth.PermissionsFromContext(ctx)
		hasGlobal := perms.IsAllowed("network/site")
		var items []models.LookupItem
		for _, g := range resp.Items {
			if hasGlobal || perms.IsAllowed("network/site/"+g.ID) {
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
	PoolsDir = "network/site/pools"
)

type SitePoolService struct {
	engine        *AnalysisEngine
	exportManager *ExportManager
	syncTasks     *taskpkg.Manager[*SyncTask]

	cron     *cron.Cron
	cronIDs  map[string]cron.EntryID
	cronLock sync.Mutex
}

func NewSitePoolService(engine *AnalysisEngine, em *ExportManager) *SitePoolService {
	s := &SitePoolService{
		engine:        engine,
		exportManager: em,
		syncTasks:     taskpkg.NewManager[*SyncTask]("action:site_sync", "sync_tasks", "network", "site"),
		cron:          cron.New(),
		cronIDs:       make(map[string]cron.EntryID),
	}
	s.cron.Start()

	// 集群事件: 变更同步策略时，刷新本节点 cron 调度 (涵盖创建、更新、删除及启停)
	common.RegisterEventHandler(common.EventSiteSyncPolicyChanged, func(ctx context.Context, policyID string) {
		policy, err := repo.GetSyncPolicy(ctx, policyID)
		if err != nil {
			s.removeCronJob(policyID)
			return
		}
		if policy.Meta.Enabled {
			s.addCronJob(*policy)
		} else {
			s.removeCronJob(policyID)
		}
	})

	// 集群事件: 异步触发同步
	common.RegisterEventHandler(common.EventSiteSyncRun, func(ctx context.Context, policyID string) {
		// 注入系统权限
		sysCtx := commonauth.WithAuth(ctx, &commonauth.AuthContext{
			Type: "sa",
			ID:   "system",
		})
		sysCtx = commonauth.WithPermissions(sysCtx, &models.ResourcePermissions{AllowedAll: true})

		go s.doSync(sysCtx, policyID)
	})

	return s
}

func (s *SitePoolService) GetSyncTaskManager() *taskpkg.Manager[*SyncTask] {
	return s.syncTasks
}

func (s *SitePoolService) Start(ctx context.Context) {
	sysCtx := commonauth.WithAuth(ctx, &commonauth.AuthContext{
		Type: "sa",
		ID:   "system",
	})
	sysCtx = commonauth.WithPermissions(sysCtx, &models.ResourcePermissions{AllowedAll: true})

	s.syncTasks.Reconcile(sysCtx)
	for _, t := range s.syncTasks.RangeAll() {
		status := t.GetStatus()
		if status == models.TaskStatusFailed || status == models.TaskStatusCancelled {
			p, err := repo.GetSyncPolicy(sysCtx, t.GetID())
			if err == nil && (p.Status.LastStatus == models.TaskStatusRunning || p.Status.LastStatus == models.TaskStatusPending) {
				p.Status.LastStatus = status
				p.Status.ErrorMessage = t.Error
				p.Status.LastRunAt = time.Now()
				_ = repo.SaveSyncPolicy(sysCtx, p)
			}
		} else if status == models.TaskStatusPending || status == models.TaskStatusRunning {
			go s.doSync(sysCtx, t.GetID())
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
}

func (s *SitePoolService) addCronJob(p models.SiteSyncPolicy) {
	s.cronLock.Lock()
	defer s.cronLock.Unlock()

	if id, ok := s.cronIDs[p.ID]; ok {
		s.cron.Remove(id)
	}

	lockKey := "site_sync_" + p.ID
	id, err := common.AddDistributedCronJob(s.cron, p.Meta.Cron, lockKey, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
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

func (s *SitePoolService) removeCronJob(id string) {
	s.cronLock.Lock()
	defer s.cronLock.Unlock()
	if entryID, ok := s.cronIDs[id]; ok {
		s.cron.Remove(entryID)
		delete(s.cronIDs, id)
	}
}
