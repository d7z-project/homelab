package intelligence

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/intelligence"
	"homelab/pkg/services/ip"
	"homelab/pkg/services/rbac"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

var (
	ErrSourceNotFound = fmt.Errorf("%w: intelligence source not found", common.ErrNotFound)
)

func init() {
	rbac.RegisterResourceWithVerbs("network/intelligence", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		return []models.DiscoverResult{}, nil
	}, []string{"list", "create", "update", "delete", "execute", "*"})
}

type IntelligenceService struct {
	mmdb    *ip.MMDBManager
	cron    *cron.Cron
	entries map[string]cron.EntryID
	mu      sync.Mutex
}

func NewIntelligenceService(mmdb *ip.MMDBManager) *IntelligenceService {
	s := &IntelligenceService{
		mmdb:    mmdb,
		cron:    cron.New(),
		entries: make(map[string]cron.EntryID),
	}
	s.cron.Start()

	// 集群事件: 其他节点更新了数据源时，本节点刷新 cron 调度
	common.RegisterEventHandler("intelligence_source_update", func(ctx context.Context, sourceID string) {
		src, err := repo.GetSource(ctx, sourceID)
		if err != nil {
			s.removeCronJob(sourceID)
			return
		}
		s.updateCronJob(*src)
	})

	common.RegisterEventHandler("intelligence_source_delete", func(ctx context.Context, sourceID string) {
		s.removeCronJob(sourceID)
	})

	return s
}

func (s *IntelligenceService) Init(ctx context.Context) error {
	sources, err := repo.ListSources(ctx)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range sources {
		src := &sources[i]
		// Reset "Downloading" status if stuck from previous run
		if src.Status == "Downloading" {
			// 健壮性：仅当该同步任务对应的分布式锁未被占有时才重置
			lockKey := "network:intelligence:sync:" + src.ID
			if release := common.Locker.TryLock(ctx, lockKey); release != nil {
				src.Status = "Error"
				src.ErrorMessage = "Interrupted by system restart or node failure"
				_ = repo.SaveSource(ctx, src)
				release()
			}
		}

		if src.AutoUpdate && src.UpdateCron != "" {
			s.addCronJob(*src)
		}
	}
	log.Printf("IntelligenceService: initialized and cleaned up stuck tasks")
	return nil
}
