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
	"time"

	"homelab/pkg/common/task"

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
	tasks   *task.Manager[*SyncTask]
}

type SyncTask struct {
	ID        string            `json:"id"`
	Status    models.TaskStatus `json:"status"`
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

var _ models.TaskInfo = (*SyncTask)(nil)

func NewIntelligenceService(mmdb *ip.MMDBManager) *IntelligenceService {
	s := &IntelligenceService{
		mmdb:    mmdb,
		cron:    cron.New(),
		entries: make(map[string]cron.EntryID),
		tasks:   task.NewManager[*SyncTask]("action:intelligence_sync", "sync_tasks", "network", "intelligence"),
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

	s.tasks.Reconcile(ctx)
	// 同步自愈后的任务状态到资源记录
	for _, t := range s.tasks.RangeAll() {
		status := t.GetStatus()
		if status == models.TaskStatusFailed || status == models.TaskStatusCancelled {
			src, err := repo.GetSource(ctx, t.GetID())
			if err == nil && (src.Status == models.TaskStatusRunning || src.Status == models.TaskStatusPending) {
				src.Status = status
				src.ErrorMessage = t.Error
				_ = repo.SaveSource(ctx, src)
			}
		}
	}

	for i := range sources {
		src := &sources[i]
		if src.AutoUpdate && src.UpdateCron != "" {
			s.addCronJob(*src)
		}
	}

	log.Printf("IntelligenceService: initialized and scheduled tasks")
	return nil
}

func (s *IntelligenceService) GetTasks() *task.Manager[*SyncTask] {
	return s.tasks
}

func (s *IntelligenceService) CancelTask(id string) bool {
	return s.tasks.CancelTask(id)
}
