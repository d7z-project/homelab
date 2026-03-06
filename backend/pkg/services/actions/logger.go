package actions

import (
	"bufio"
	"fmt"
	"homelab/pkg/models"
	"os"
	"path"
	"sync"
	"time"

	"github.com/spf13/afero"
)

type TaskLogger struct {
	mu           sync.Mutex
	workflowID   string
	instanceID   string
	currentIndex int
	currentFile  afero.File
}

// NewTaskLogger creates a new logger that writes to a file in logFS.
// Initial file is index 0 (engine logs).
func NewTaskLogger(workflowID, instanceID string) (*TaskLogger, error) {
	l := &TaskLogger{
		workflowID:   workflowID,
		instanceID:   instanceID,
		currentIndex: 0,
	}
	if err := l.openCurrent(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *TaskLogger) getLogDir() string {
	return path.Join("actions", l.workflowID, l.instanceID)
}

func (l *TaskLogger) openCurrent() error {
	if l.currentFile != nil {
		_ = l.currentFile.Close()
	}

	dir := l.getLogDir()
	_ = logFS.MkdirAll(dir, 0755)

	filename := path.Join(dir, fmt.Sprintf("%d.log", l.currentIndex))
	// Open in append mode, create if not exists
	f, err := logFS.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", filename, err)
	}
	l.currentFile = f
	return nil
}

// SetStep switches to a new log file for a specific step index.
func (l *TaskLogger) SetStep(index int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.currentIndex = index
	_ = l.openCurrent()
}

// Log writes a raw text line with a timestamp.
func (l *TaskLogger) Log(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentFile == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	_, _ = fmt.Fprintf(l.currentFile, "[%s] %s\n", timestamp, message)
}

func (l *TaskLogger) Logf(format string, a ...interface{}) {
	l.Log(fmt.Sprintf(format, a...))
}

func getReadLogPath(workflowID, instanceID string, index int) string {
	return path.Join("actions", workflowID, instanceID, fmt.Sprintf("%d.log", index))
}

// ReadStepLogs reads logs for a specific step index starting from a line offset.
func ReadStepLogs(workflowID, instanceID string, index int, offset int) ([]models.LogEntry, int, error) {
	filename := getReadLogPath(workflowID, instanceID, index)
	f, err := logFS.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return []models.LogEntry{}, 0, nil
		}
		return nil, 0, err
	}
	defer f.Close()

	var logs []models.LogEntry
	scanner := bufio.NewScanner(f)
	// Increase buffer size to handle large lines (up to 1MB)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	currentLine := 0
	for scanner.Scan() {
		if currentLine >= offset {
			line := scanner.Text()
			entry := models.LogEntry{
				Message: line,
			}
			// Attempt to parse timestamp if it looks like "[YYYY-MM-DD HH:MM:SS] "
			if len(line) > 21 && line[0] == '[' && line[20] == ']' {
				tsStr := line[1:20]
				if ts, err := time.ParseInLocation("2006-01-02 15:04:05", tsStr, time.Local); err == nil {
					entry.Timestamp = ts
					entry.Message = line[22:]
				}
			}
			logs = append(logs, entry)
		}
		currentLine++
	}
	return logs, currentLine, scanner.Err()
}

// ReadAllTaskLogs aggregates logs from all step files for an instance.
func ReadAllTaskLogs(workflowID, instanceID string) ([]models.LogEntry, error) {
	var allLogs []models.LogEntry

	// We scan files sequentially from index 0 until we hit a gap or error
	for i := 0; ; i++ {
		logs, _, err := ReadStepLogs(workflowID, instanceID, i, 0)
		if err != nil || len(logs) == 0 {
			// Check if the file exists but is empty vs doesn't exist
			filename := getReadLogPath(workflowID, instanceID, i)
			if _, statErr := logFS.Stat(filename); statErr != nil {
				break // Stop if file doesn't exist
			}
		}
		allLogs = append(allLogs, logs...)

		// Safety break to prevent infinite loops if something goes wrong
		if i > 1000 {
			break
		}
	}
	return allLogs, nil
}

// RemoveTaskLogs cleans up all log files associated with an instance.
func RemoveTaskLogs(workflowID, instanceID string) error {
	dir := path.Join("actions", workflowID, instanceID)
	return logFS.RemoveAll(dir)
}

// RemoveWorkflowLogs cleans up all logs associated with a workflow.
func RemoveWorkflowLogs(workflowID string) error {
	return logFS.RemoveAll(path.Join("actions", workflowID))
}

func (l *TaskLogger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.currentFile != nil {
		_ = l.currentFile.Close()
	}
}
