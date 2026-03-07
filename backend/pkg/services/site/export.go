package site

import (
	"context"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/common/task"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/site"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/expr-lang/expr"
)

type ExportTask struct {
	ID          string    `json:"ID"`
	Status      string    `json:"Status"` // Pending, Running, Success, Failed, Cancelled
	Progress    float64   `json:"Progress"`
	Format      string    `json:"Format"`
	ResultURL   string    `json:"ResultURL"`
	Error       string    `json:"Error"`
	CreatedAt   time.Time `json:"CreatedAt"`
	RecordCount int64     `json:"RecordCount"`
	mu          sync.Mutex
}

func (t *ExportTask) GetID() string { return t.ID }
func (t *ExportTask) GetStatus() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Status
}
func (t *ExportTask) SetStatus(status string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status
}
func (t *ExportTask) SetError(msg string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Error = msg
}
func (t *ExportTask) GetCreatedAt() time.Time { return t.CreatedAt }

func (t *ExportTask) MarshalJSON() ([]byte, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return json.Marshal(ExportTaskDTO{
		ID:          t.ID,
		Status:      t.Status,
		Progress:    t.Progress,
		Format:      t.Format,
		ResultURL:   t.ResultURL,
		Error:       t.Error,
		CreatedAt:   t.CreatedAt,
		RecordCount: t.RecordCount,
	})
}

var _ models.TaskInfo = (*ExportTask)(nil)

type ExportTaskDTO struct {
	ID          string    `json:"ID"`
	Status      string    `json:"Status"`
	Progress    float64   `json:"Progress"`
	Format      string    `json:"Format"`
	ResultURL   string    `json:"ResultURL"`
	Error       string    `json:"Error"`
	CreatedAt   time.Time `json:"CreatedAt"`
	RecordCount int64     `json:"RecordCount"`
}

type ExportManager struct {
	core     *task.Manager[*ExportTask]
	analysis *AnalysisEngine
	wg       sync.WaitGroup
}

func NewExportManager(analysis *AnalysisEngine) *ExportManager {
	core := task.NewManager[*ExportTask]("action:site_export", "export_tasks", "network", "site")

	core.SetCleanupHook(func(t *ExportTask) {
		tempFileName := fmt.Sprintf("site_export_%s.%s", t.ID, t.Format)
		tempPath := filepath.Join("temp", tempFileName)
		_ = common.TempDir.Remove(tempPath)
	})

	core.StartCleanupTimer(24*time.Hour, 1*time.Hour)

	return &ExportManager{
		core:     core,
		analysis: analysis,
	}
}

func (m *ExportManager) Reconcile(ctx context.Context) {
	m.core.Reconcile(ctx)
}

func (m *ExportManager) GetTask(id string) *ExportTaskDTO {
	t, ok := m.core.GetTask(id)
	if !ok {
		return nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return &ExportTaskDTO{
		ID:          t.ID,
		Status:      t.Status,
		Progress:    t.Progress,
		Format:      t.Format,
		ResultURL:   t.ResultURL,
		Error:       t.Error,
		CreatedAt:   t.CreatedAt,
		RecordCount: t.RecordCount,
	}
}

func (m *ExportManager) ListTasks() []ExportTaskDTO {
	tasks := m.core.RangeAll()
	var res []ExportTaskDTO
	for _, t := range tasks {
		t.mu.Lock()
		res = append(res, ExportTaskDTO{
			ID:          t.ID,
			Status:      t.Status,
			Progress:    t.Progress,
			Format:      t.Format,
			ResultURL:   t.ResultURL,
			Error:       t.Error,
			CreatedAt:   t.CreatedAt,
			RecordCount: t.RecordCount,
		})
		t.mu.Unlock()
	}
	return res
}

func (m *ExportManager) DeleteTasksByExportID(exportID string) {
	m.core.DeleteTasksByPrefix(exportID + "-")
}

func (m *ExportManager) TriggerExport(ctx context.Context, exportID string, format string) (string, error) {
	e, err := repo.GetExport(ctx, exportID)
	if err != nil {
		return "", err
	}

	tasks := m.core.RangeAll()
	var toCancel []string
	for _, t := range tasks {
		if strings.HasPrefix(t.ID, exportID+"-") {
			status := t.GetStatus()
			if status == "Pending" || status == "Running" {
				lockKey := "action:site_export:" + t.ID
				if release := common.Locker.TryLock(ctx, lockKey); release != nil {
					toCancel = append(toCancel, t.ID)
					release()
				} else {
					return "", fmt.Errorf("an export task for %s is already in progress", exportID)
				}
			}
		}
	}

	for _, id := range toCancel {
		m.core.CancelTask(id)
	}

	taskID := fmt.Sprintf("%s-%d", exportID, time.Now().UnixNano())
	task := &ExportTask{
		ID:        taskID,
		Status:    "Pending",
		Format:    format,
		CreatedAt: time.Now(),
	}

	m.core.AddTask(task)

	m.wg.Add(1)
	go m.runExport(context.Background(), taskID, e)

	return taskID, nil
}

func (m *ExportManager) WaitAll() {
	m.wg.Wait()
}

func (m *ExportManager) runExport(bgCtx context.Context, taskID string, e *models.SiteExport) {
	defer m.wg.Done()

	m.core.RunTask(bgCtx, taskID, func(taskCtx context.Context, task *ExportTask) error {
		program, err := expr.Compile(e.Rule, expr.Env(map[string]interface{}{
			"tags":   []string{},
			"domain": "",
			"type":   uint8(0),
		}))
		if err != nil {
			return fmt.Errorf("Compile error: %w", err)
		}

		totalEntries := int64(0)
		for _, gid := range e.GroupIDs {
			g, _ := repo.GetGroup(taskCtx, gid)
			if g != nil {
				totalEntries += g.EntryCount
			}
		}
		if totalEntries == 0 {
			totalEntries = 1
		}

		tempFileName := fmt.Sprintf("site_export_%s.%s", task.ID, task.Format)
		tempPath := filepath.Join("temp", tempFileName)
		_ = common.TempDir.MkdirAll("temp", 0755)
		f, err := common.TempDir.Create(tempPath)
		if err != nil {
			return fmt.Errorf("File create error: %w", err)
		}
		defer f.Close()

		if task.Format == "json" {
			f.WriteString("[\n")
		}
		firstItem := true
		totalRead := int64(0)

		for _, gid := range e.GroupIDs {
			poolPath := filepath.Join(PoolsDir, gid+".bin")
			pf, err := common.FS.Open(poolPath)
			if err != nil {
				continue
			}
			reader, err := NewReader(pf)
			if err != nil {
				pf.Close()
				continue
			}

			for {
				select {
				case <-taskCtx.Done():
					pf.Close()
					return context.Canceled
				default:
				}

				entry, err := reader.Next()
				if err == io.EOF {
					break
				}
				totalRead++

				out, err := expr.Run(program, map[string]interface{}{
					"tags":   entry.Tags,
					"domain": entry.Value,
					"type":   entry.Type,
				})

				if err == nil && out == true {
					task.mu.Lock()
					task.RecordCount++
					task.mu.Unlock()

					if !firstItem && task.Format == "json" {
						f.WriteString(",\n")
					}
					writeEntry(f, task.Format, entry)
					firstItem = false
				}

				if totalRead%1000 == 0 {
					task.mu.Lock()
					task.Progress = float64(totalRead) / float64(totalEntries)
					task.mu.Unlock()
					m.core.Save()
				}
			}
			pf.Close()
		}

		if task.Format == "json" {
			f.WriteString("\n]\n")
		}

		task.mu.Lock()
		task.Progress = 1.0
		task.ResultURL = "/api/v1/network/site/exports/download/" + task.ID
		task.mu.Unlock()
		return nil
	})
}

func writeEntry(w io.Writer, format string, e models.SitePoolEntry) {
	var publicTags []string
	for _, t := range e.Tags {
		if !strings.HasPrefix(t, "_") {
			publicTags = append(publicTags, t)
		}
	}
	e.Tags = publicTags

	switch format {
	case "text":
		prefix := ""
		switch e.Type {
		case 0:
			prefix = "keyword:"
		case 1:
			prefix = "regexp:"
		case 2:
			prefix = "domain:"
		case 3:
			prefix = "full:"
		}
		fmt.Fprintf(w, "%s%s\n", prefix, e.Value)
	case "json":
		b, _ := json.Marshal(e)
		w.Write(b)
	case "yaml":
		fmt.Fprintf(w, "- type: %d\n  value: %s\n", e.Type, e.Value)
		if len(e.Tags) > 0 {
			fmt.Fprintln(w, "  tags:")
			for _, t := range e.Tags {
				fmt.Fprintf(w, "    - %s\n", t)
			}
		}
	}
}
