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
	// 集群事件: 变更工作流触发器时，刷新本节点 cron 调度 (涵盖创建、更新、删除及启停)
	common.RegisterEventHandler(common.EventWorkflowTriggerChanged, func(ctx context.Context, workflowID string) {
		wf, err := repo.GetWorkflow(ctx, workflowID)
		if err != nil {
			m.RemoveTriggers(workflowID)
			return
		}
		m.UpdateTriggers(*wf)
	})

	// 异步执行工作流事件
	common.RegisterEventHandler(common.EventWorkflowExecute, func(ctx context.Context, req models.WorkflowExecutePayload) {
		// 使用分布式锁确保同一实例 ID 只被一个节点执行
		lockKey := "action:execute:" + req.InstanceID
		release := common.Locker.TryLock(ctx, lockKey)
		if release == nil {
			// 锁被占用，说明已有节点在执行
			return
		}
		// 注意：Execute 内部会管理自己的生命周期，这里的 release 仅用于抢占权
		// 我们不需要在这里 defer release()，因为 Execute 异步启动后，抢占权已完成
		// 实际上，为了防止多节点竞争，拿到锁即视为“领任务成功”
		// 但为了安全，我们可以保持锁一段时间，或者相信 Execute 内部的 local/global concurrency check

		wf, err := repo.GetWorkflow(ctx, req.WorkflowID)
		if err != nil {
			log.Printf("TriggerManager: failed to fetch workflow %s for async execution: %v", req.WorkflowID, err)
			return
		}

		log.Printf("TriggerManager: picking up async execution for instance %s", req.InstanceID)
		_, err = GlobalExecutor.Execute(ctx, req.UserID, wf, req.Trigger, req.Inputs, req.InstanceID)
		if err != nil {
			log.Printf("TriggerManager: failed to start async execution for instance %s: %v", req.InstanceID, err)
		}
	})
}

func (m *TriggerManager) InitTriggers(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	workflows, err := repo.ScanAllWorkflows(ctx)
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
