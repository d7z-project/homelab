package task

import (
	"sync"
	"time"

	"homelab/pkg/models/shared"
)

// SimpleTask is the shared task state container used by sync-style background jobs.
type SimpleTask struct {
	ID        string            `json:"id"`
	Status    shared.TaskStatus `json:"status"`
	Progress  float64           `json:"progress"`
	Error     string            `json:"error"`
	CreatedAt time.Time         `json:"createdAt"`
	mu        sync.Mutex
}

func (t *SimpleTask) GetID() string { return t.ID }

func (t *SimpleTask) GetStatus() shared.TaskStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Status
}

func (t *SimpleTask) SetStatus(status shared.TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status
}

func (t *SimpleTask) SetError(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Error = msg
}

func (t *SimpleTask) GetCreatedAt() time.Time { return t.CreatedAt }

func (t *SimpleTask) GetProgress() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Progress
}

func (t *SimpleTask) SetProgress(progress float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Progress = progress
}

var _ shared.TaskInfo = (*SimpleTask)(nil)
