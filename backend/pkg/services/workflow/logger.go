package workflow

import (
	"bufio"
	"context"
	"fmt"
	"homelab/pkg/common"
	"os"
	"path"
	"sync"
	"time"

	workflowmodel "homelab/pkg/models/workflow"

	"github.com/spf13/afero"
)

type TaskLogger struct {
	mu           sync.Mutex
	workflowID   string
	instanceID   string
	currentIndex int
	currentFile  afero.File
	lineCount    int
}

// NewTaskLogger creates a new logger that writes to a temporary file in logFS.
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
		l.currentFile = nil
		l.promoteTmpFile(l.currentIndex)
	}

	dir := l.getLogDir()
	_ = logFS.MkdirAll(dir, 0755)

	tmpFilename := path.Join(dir, fmt.Sprintf("%d.log.tmp", l.currentIndex))
	// Open in append mode, create if not exists
	f, err := logFS.OpenFile(tmpFilename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", tmpFilename, err)
	}
	l.currentFile = f
	l.lineCount = 0
	return nil
}

func (l *TaskLogger) promoteTmpFile(index int) {
	oldPath := path.Join(l.getLogDir(), fmt.Sprintf("%d.log.tmp", index))
	newPath := path.Join(l.getLogDir(), fmt.Sprintf("%d.log", index))
	if exists, _ := afero.Exists(logFS, oldPath); exists {
		// Acquire distributed lock for final rename
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		lockKey := fmt.Sprintf("actions:log_rename:%s:%s:%d", l.workflowID, l.instanceID, index)
		release := common.Locker.TryLock(ctx, lockKey)
		if release != nil {
			defer release()
			_ = logFS.Rename(oldPath, newPath)
		}
	}
}

// SetStep switches to a new log file for a specific step index.
func (l *TaskLogger) SetStep(index int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Close current and rename if changed
	if l.currentIndex != index {
		prevIndex := l.currentIndex
		l.currentIndex = index
		_ = l.openCurrent()
		l.promoteTmpFile(prevIndex)
	}
}

// Log writes a raw text line with a timestamp.
func (l *TaskLogger) Log(message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.currentFile == nil {
		return
	}

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formatted := fmt.Sprintf("[%s] %s\n", timestamp, message)
	_, _ = l.currentFile.WriteString(formatted)

	// Explicitly flush to make logs visible to other nodes watching VFS
	if syncer, ok := l.currentFile.(interface{ Sync() error }); ok {
		_ = syncer.Sync()
	}

	// Distributed temporary storage in DB for real-time aggregation across nodes
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	key := fmt.Sprintf("%08d", l.lineCount)
	l.lineCount++
	if common.DB != nil {
		_ = common.DB.Child("system", "task:logs", l.instanceID, fmt.Sprint(l.currentIndex)).Put(ctx, key, formatted, 24*time.Hour)
	}
}

func (l *TaskLogger) Logf(format string, a ...interface{}) {
	l.Log(fmt.Sprintf(format, a...))
}

func getReadLogPath(workflowID, instanceID string, index int) string {
	return path.Join("actions", workflowID, instanceID, fmt.Sprintf("%d.log", index))
}

func getReadLogPathTmp(workflowID, instanceID string, index int) string {
	return path.Join("actions", workflowID, instanceID, fmt.Sprintf("%d.log.tmp", index))
}

// ReadStepLogs reads logs for a specific step index starting from a line offset.
func ReadStepLogs(workflowID, instanceID string, index int, offset int) ([]workflowmodel.LogEntry, int, error) {
	filename := getReadLogPath(workflowID, instanceID, index)
	f, err := logFS.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			// Fallback to read from .tmp stream file if final log is missing
			tmpFilename := getReadLogPathTmp(workflowID, instanceID, index)
			f, err = logFS.Open(tmpFilename)
			if err != nil {
				if os.IsNotExist(err) {
					return []workflowmodel.LogEntry{}, 0, nil
				}
				return nil, 0, err
			}
		} else {
			return nil, 0, err
		}
	}
	defer f.Close()

	var logs []workflowmodel.LogEntry
	scanner := bufio.NewScanner(f)
	// Increase buffer size to handle large lines (up to 1MB)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)
	currentLine := 0
	for scanner.Scan() {
		if currentLine >= offset {
			line := scanner.Text()
			entry := workflowmodel.LogEntry{
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
func ReadAllTaskLogs(workflowID, instanceID string) ([]workflowmodel.LogEntry, error) {
	var allLogs []workflowmodel.LogEntry

	// We scan files sequentially from index 0 until we hit a gap or error
	for i := 0; ; i++ {
		logs, _, err := ReadStepLogs(workflowID, instanceID, i, 0)
		if err != nil || len(logs) == 0 {
			// Check if the file (or tmp file) exists but is empty vs doesn't exist
			filename := getReadLogPath(workflowID, instanceID, i)
			tmpFilename := getReadLogPathTmp(workflowID, instanceID, i)
			_, statErr := logFS.Stat(filename)
			_, tmpStatErr := logFS.Stat(tmpFilename)

			if statErr != nil && tmpStatErr != nil {
				break // Stop if both files don't exist
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
	_ = common.DB.Child("system", "task:logs", instanceID).DeleteAll(context.Background())
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
		l.currentFile = nil
		l.promoteTmpFile(l.currentIndex)
	}
}
