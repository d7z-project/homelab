package actions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/actions"
	"log"
	"sync"

	"github.com/robfig/cron/v3"
)

type TriggerManager struct {
	cron    *cron.Cron
	entries map[string]cron.EntryID // workflowID -> entryID
	mu      sync.Mutex
}

var GlobalTriggerManager = &TriggerManager{
	cron:    cron.New(),
	entries: make(map[string]cron.EntryID),
}

func (m *TriggerManager) Start() {
	m.registerClusterHandlers()
	m.cron.Start()
}

func (m *TriggerManager) registerClusterHandlers() {
	// 当其他节点创建/更新了工作流时，本节点也需要刷新 cron 调度
	common.RegisterEventHandler("workflow_trigger_update", func(ctx context.Context, workflowID string) {
		wf, err := repo.GetWorkflow(ctx, workflowID)
		if err != nil {
			return
		}
		m.UpdateTriggers(*wf)
	})

	// 当其他节点删除了工作流时，本节点也需要移除 cron 调度
	common.RegisterEventHandler("workflow_trigger_delete", func(ctx context.Context, workflowID string) {
		m.RemoveTriggers(workflowID)
	})
}

func (m *TriggerManager) InitTriggers(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	workflows, err := repo.ListWorkflows(ctx)
	if err != nil {
		return err
	}

	count := 0
	for _, wf := range workflows {
		if wf.Enabled && wf.CronEnabled && wf.CronExpr != "" {
			m.addCronJob(wf)
			count++
		}
	}
	log.Printf("TriggerManager: restored %d cron jobs from persistence", count)
	return nil
}

func (m *TriggerManager) UpdateTriggers(wf models.Workflow) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Remove existing if any
	if entryID, ok := m.entries[wf.ID]; ok {
		m.cron.Remove(entryID)
		delete(m.entries, wf.ID)
	}

	// Add new if enabled
	if wf.Enabled && wf.CronEnabled && wf.CronExpr != "" {
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

func (m *TriggerManager) addCronJob(wf models.Workflow) {
	lockKey := "workflow_cron_" + wf.ID
	id := wf.ID
	entryID, err := common.AddDistributedCronJob(m.cron, wf.CronExpr, lockKey, func() {
		// fetch latest workflow from db to avoid stale struct capture and missed updates
		latestWf, err := repo.GetWorkflow(context.Background(), id)
		if err != nil || !latestWf.Enabled || !latestWf.CronEnabled {
			return
		}

		log.Printf("Triggering cron job for workflow: %s (%s)", latestWf.Name, latestWf.ID)
		// Use the configured ServiceAccount for execution
		_, err = TriggerWorkflow(context.Background(), latestWf, latestWf.ServiceAccountID, "Cron", nil)
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
