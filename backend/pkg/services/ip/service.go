package ip

import (
	"context"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
	"homelab/pkg/services/discovery"
	"homelab/pkg/services/rbac"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

func init() {
	rbac.RegisterResourceWithVerbs("network/ip", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		res := make([]models.DiscoverResult, 0)
		groups, _, err := repo.ListGroups(ctx, 1, 1000, "")
		if err != nil {
			return nil, err
		}
		for _, g := range groups {
			if strings.HasPrefix(g.ID, prefix) || strings.HasPrefix(g.Name, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: g.ID,
					Name:   g.Name,
					Final:  true,
				})
			}
		}
		return res, nil
	}, []string{"get", "list", "create", "update", "delete", "execute", "*"})

	discovery.Register("network/ip/pools", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		groups, _, err := repo.ListGroups(ctx, 1, 1000, search)
		if err != nil {
			return nil, 0, err
		}
		var items []models.LookupItem
		for _, g := range groups {
			items = append(items, models.LookupItem{
				ID:          g.ID,
				Name:        g.Name,
				Description: g.Description,
			})
		}
		total := len(items)
		if limit <= 0 {
			limit = 20
		}
		if offset >= total {
			return []models.LookupItem{}, total, nil
		}
		end := offset + limit
		if end > total {
			end = total
		}
		return items[offset:end], total, nil
	})
}

const (
	PoolsDir       = "network/ip/pools"
	MaxPoolEntries = 2000000
)

type IPPoolService struct {
	mmdb           *MMDBManager
	cron           *cron.Cron
	cronIDs        map[string]cron.EntryID
	cronLock       sync.Mutex
	exportManager  *ExportManager
	analysisEngine *AnalysisEngine
}

func NewIPPoolService(mmdb *MMDBManager) *IPPoolService {
	svc := &IPPoolService{
		mmdb:    mmdb,
		cron:    cron.New(),
		cronIDs: make(map[string]cron.EntryID),
	}

	// 集群事件: 其他节点更新了同步策略时，本节点刷新 cron 调度
	common.RegisterEventHandler("ip_sync_policy_update", func(ctx context.Context, policyID string) {
		policy, err := repo.GetSyncPolicy(ctx, policyID)
		if err != nil {
			svc.removeCronJob(policyID)
			return
		}
		if policy.Enabled {
			svc.addCronJob(*policy)
		} else {
			svc.removeCronJob(policyID)
		}
	})

	common.RegisterEventHandler("ip_sync_policy_delete", func(ctx context.Context, policyID string) {
		svc.removeCronJob(policyID)
	})

	// 集群事件: 异步触发同步
	common.RegisterEventHandler("ip_sync_run", func(ctx context.Context, policyID string) {
		// 使用分布式锁确保同一策略只被一个节点执行
		lockKey := "action:ip_sync:" + policyID
		release := common.Locker.TryLock(ctx, lockKey)
		if release == nil {
			return
		}
		defer release()

		// 注入系统权限
		sysCtx := commonauth.WithAuth(ctx, &commonauth.AuthContext{
			Type: "sa",
			ID:   "system",
		})
		sysCtx = commonauth.WithPermissions(sysCtx, &models.ResourcePermissions{AllowedAll: true})

		policy, err := repo.GetSyncPolicy(sysCtx, policyID)
		if err != nil || !policy.Enabled {
			return
		}

		_ = svc.doSync(sysCtx, policy)
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

func (s *IPPoolService) SetExportManager(em *ExportManager) {
	s.exportManager = em
}

func (s *IPPoolService) SetAnalysisEngine(ae *AnalysisEngine) {
	s.analysisEngine = ae
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

	// 1. 状态自愈 (Reconciliation)
	// 扫描所有运行中的策略，如果能拿到对应的分布式锁，说明该任务已僵死，需要重置
	policies, _, _ := repo.ListSyncPolicies(sysCtx, 1, 10000, "")
	for _, p := range policies {
		if p.LastStatus == "running" || p.LastStatus == "pending" {
			lockKey := "action:ip_sync:" + p.ID
			if release := common.Locker.TryLock(sysCtx, lockKey); release != nil {
				// 任务僵死：更新状态并释放锁
				p.LastStatus = "failed"
				p.ErrorMessage = "interrupted by system restart"
				_ = repo.SaveSyncPolicy(sysCtx, &p)
				release()
			}
		}

		if p.Enabled {
			s.addCronJob(p)
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
	id, err := common.AddDistributedCronJob(s.cron, p.Cron, lockKey, func() {
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
