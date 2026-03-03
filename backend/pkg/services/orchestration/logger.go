package orchestration

import (
	"fmt"
	"homelab/pkg/models"
	"sync"
	"time"
)

type TaskLogger struct {
	mu          sync.Mutex
	logs        []models.LogEntry
	currentStep string
}

func NewTaskLogger() *TaskLogger {
	return &TaskLogger{
		logs: make([]models.LogEntry, 0),
	}
}

func (l *TaskLogger) SetStep(stepID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.currentStep = stepID
}

func (l *TaskLogger) Log(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.logs = append(l.logs, models.LogEntry{
		Timestamp: time.Now(),
		StepID:    l.currentStep,
		Message:   message,
	})
}

func (l *TaskLogger) Logf(format string, a ...interface{}) {
	l.Log(fmt.Sprintf(format, a...))
}

func (l *TaskLogger) GetLogs() []models.LogEntry {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Return a copy to avoid race conditions
	res := make([]models.LogEntry, len(l.logs))
	copy(res, l.logs)
	return res
}

func (l *TaskLogger) Close() {
	// No-op for now as we don't have external resources to close
}
