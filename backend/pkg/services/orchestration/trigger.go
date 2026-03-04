package orchestration

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/orchestration"
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
	m.cron.Start()
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
	entryID, err := m.cron.AddFunc(wf.CronExpr, func() {
		log.Printf("Triggering cron job for workflow: %s (%s)", wf.Name, wf.ID)
		// Use the configured ServiceAccount for execution
		_, err := TriggerWorkflow(context.Background(), &wf, wf.ServiceAccountID, "Cron", nil)
		if err != nil {
			log.Printf("Failed to trigger cron job for %s: %v", wf.ID, err)
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
