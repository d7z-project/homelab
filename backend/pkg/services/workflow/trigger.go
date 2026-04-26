package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"homelab/pkg/common"
	workflowmodel "homelab/pkg/models/workflow"
	repo "homelab/pkg/repositories/workflow/actions"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

type TriggerManager struct {
	runtime *Runtime
	cron    *cron.Cron
	entries map[string]cron.EntryID // workflowID -> entryID
	mu      sync.Mutex
}

func NewTriggerManager(rt *Runtime) *TriggerManager {
	return &TriggerManager{
		runtime: rt,
		cron:    cron.New(),
		entries: make(map[string]cron.EntryID),
	}
}

func (m *TriggerManager) Start(ctx context.Context) {
	m.registerClusterHandlers(ctx)
	m.cron.Start()
}

func (m *TriggerManager) registerClusterHandlers(ctx context.Context) {
	rt := m.runtime
	// 集群事件: 变更工作流触发器时，刷新本节点 cron 调度 (涵盖创建、更新、删除及启停)
	common.RegisterEventHandler(common.EventWorkflowTriggerChanged, func(ctx context.Context, payload common.ResourceEventPayload) {
		ctx = rt.WithContext(ctx)
		wf, err := repo.GetWorkflow(ctx, payload.ID)
		if err != nil {
			m.RemoveTriggers(payload.ID)
			return
		}
		m.UpdateTriggers(*wf)
	})
}

func (m *TriggerManager) InitTriggers(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	cursor := ""
	for {
		workflows, err := repo.ScanWorkflows(ctx, cursor, 200, "")
		if err != nil {
			return err
		}
		for _, wf := range workflows.Items {
			if wf.Meta.Enabled && wf.Meta.CronEnabled && wf.Meta.CronExpr != "" {
				m.addCronJob(wf)
				count++
			}
		}
		if !workflows.HasMore {
			break
		}
		cursor = workflows.NextCursor
	}
	log.Printf("TriggerManager: restored %d cron jobs from persistence", count)
	return nil
}

func (m *TriggerManager) UpdateTriggers(wf workflowmodel.Workflow) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove existing if any
	if entryID, ok := m.entries[wf.ID]; ok {
		m.cron.Remove(entryID)
		delete(m.entries, wf.ID)
	}

	// Add new if enabled
	if wf.Meta.Enabled && wf.Meta.CronEnabled && wf.Meta.CronExpr != "" {
		m.addCronJob(wf)
	}
}

func (m *TriggerManager) RemoveTriggers(workflowID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if entryID, ok := m.entries[workflowID]; ok {
		m.cron.Remove(entryID)
		delete(m.entries, workflowID)
	}
}

func (m *TriggerManager) addCronJob(wf workflowmodel.Workflow) {
	lockKey := "workflow_cron_" + wf.ID
	id := wf.ID
	entryID, err := common.AddDistributedCronJob(m.cron, wf.Meta.CronExpr, lockKey, func() {
		// fetch latest workflow from db to avoid stale struct capture and missed updates
		ctx := m.runtime.WithContext(context.Background())
		latestWf, err := repo.GetWorkflow(ctx, id)
		if err != nil || !latestWf.Meta.Enabled || !latestWf.Meta.CronEnabled {
			return
		}

		log.Printf("Triggering cron job for workflow: %s (%s)", latestWf.Meta.Name, latestWf.ID)
		// Use the configured ServiceAccount for execution
		_, err = TriggerWorkflow(ctx, latestWf, latestWf.Meta.ServiceAccountID, "Cron", nil)
		if err != nil {
			log.Printf("Failed to trigger cron job for %s: %v", latestWf.ID, err)
		}
	})

	if err != nil {
		log.Printf("Error scheduling cron for workflow %s: %v", wf.ID, err)
		return
	}
	m.entries[wf.ID] = entryID
}

func GenerateWebhookToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}
