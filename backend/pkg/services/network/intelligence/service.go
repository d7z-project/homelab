package intelligence

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	intelligencemodel "homelab/pkg/models/network/intelligence"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/intelligence"
	"homelab/pkg/services/network/ip"
	"log"
	"sync"
	"time"

	"homelab/pkg/common/task"

	"github.com/robfig/cron/v3"
)

var (
	ErrSourceNotFound = fmt.Errorf("%w: intelligence source not found", common.ErrNotFound)
)

type IntelligenceService struct {
	mmdb    *ip.MMDBManager
	cron    *cron.Cron
	entries map[string]cron.EntryID
	mu      sync.Mutex
	tasks   *task.Manager[*SyncTask]
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

func NewIntelligenceService(mmdb *ip.MMDBManager) *IntelligenceService {
	s := &IntelligenceService{
		mmdb:    mmdb,
		cron:    cron.New(),
		entries: make(map[string]cron.EntryID),
		tasks:   task.NewManager[*SyncTask]("action:intelligence_sync", "sync_tasks", "network", "intelligence"),
	}
	s.cron.Start()

	// 集群事件: 变更数据源时，刷新本节点 cron 调度 (涵盖创建、更新、删除及启停)
	common.RegisterEventHandler(common.EventIntelligenceSourceChanged, func(ctx context.Context, sourceID string) {
		src, err := repo.SourceRepo.Get(ctx, sourceID)
		if err != nil {
			s.removeCronJob(sourceID)
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
				_ = repo.SourceRepo.Cow(ctx, src.ID, func(res *intelligencemodel.IntelligenceSource) error {
					res.Meta = src.Meta
					res.Status = src.Status
					return nil
				})
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
	return nil
}

func (s *IntelligenceService) GetTasks() *task.Manager[*SyncTask] {
	return s.tasks
}

func (s *IntelligenceService) CancelTask(id string) bool {
	return s.tasks.CancelTask(id)
}
