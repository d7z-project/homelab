package orchestration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"homelab/pkg/models"
	"sync"
	"time"

	"github.com/spf13/afero"
)

type TaskLogger struct {
	mu          sync.Mutex
	currentStep string
	file        afero.File
	instanceID  string
}

// NewTaskLogger creates a new logger that writes to a file in logFS.
func NewTaskLogger(instanceID string) (*TaskLogger, error) {
	filename := fmt.Sprintf("%s.log", instanceID)
	f, err := logFS.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}
	return &TaskLogger{
		instanceID: instanceID,
		file:       f,
	}, nil
}

func (l *TaskLogger) SetStep(stepID string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.currentStep = stepID
}

func (l *TaskLogger) Log(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry := models.LogEntry{
		Timestamp: time.Now(),
		StepID:    l.currentStep,
		Message:   message,
	}

	data, err := json.Marshal(entry)
	if err == nil {
		_, _ = l.file.Write(data)
		_, _ = l.file.Write([]byte("\n"))
	}
}

func (l *TaskLogger) Logf(format string, a ...interface{}) {
	l.Log(fmt.Sprintf(format, a...))
}

// ReadTaskLogs reads and parses logs from the VFS file for a given instance.
func ReadTaskLogs(instanceID string) ([]models.LogEntry, error) {
	filename := fmt.Sprintf("%s.log", instanceID)
	f, err := logFS.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	var logs []models.LogEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var entry models.LogEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err == nil {
			logs = append(logs, entry)
		}
	}
	return logs, scanner.Err()
}

func (l *TaskLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		_ = l.file.Close()
	}
}
