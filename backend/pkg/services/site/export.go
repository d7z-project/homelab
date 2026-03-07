package site

import (
	"context"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/site"
	"io"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"gopkg.d7z.net/middleware/kv"
)

type ExportTask struct {
	ID          string
	Status      string // Pending, Running, Success, Failed, Cancelled
	Progress    float64
	Format      string
	ResultURL   string
	Error       string
	CreatedAt   time.Time
	RecordCount int64
	mu          sync.Mutex
}

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
	mu       sync.RWMutex
	tasks    map[string]*ExportTask
	analysis *AnalysisEngine
	wg       sync.WaitGroup
}

func NewExportManager(analysis *AnalysisEngine) *ExportManager {
	m := &ExportManager{
		tasks:    make(map[string]*ExportTask),
		analysis: analysis,
	}
	m.loadTasks()
	return m
}

func (m *ExportManager) loadTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := common.DB.Child("network", "site").Get(context.Background(), "export_tasks")
	if err == nil && data != "" {
		_ = json.Unmarshal([]byte(data), &m.tasks)
		// 注意：此处不再盲目重置状态，交给 Reconcile 处理
	}
}

// Reconcile 扫描所有任务，清理僵死状态
func (m *ExportManager) Reconcile(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	changed := false
	for _, t := range m.tasks {
		t.mu.Lock()
		if t.Status == "Running" || t.Status == "Pending" {
			// 探测锁状态
			lockKey := "action:site_export:" + t.ID
			if release := common.Locker.TryLock(ctx, lockKey); release != nil {
				// 能拿到锁说明执行者已挂
				t.Status = "Failed"
				t.Error = "Interrupted by system restart or node failure"
				changed = true
				release()
			}
		}
		t.mu.Unlock()
	}
	if changed {
		m.saveTasksLocked()
	}
}

func (m *ExportManager) saveTasksLocked() {
	dumps := make(map[string]ExportTaskDTO)
	for id, t := range m.tasks {
		t.mu.Lock()
		dumps[id] = ExportTaskDTO{
			ID:          t.ID,
			Status:      t.Status,
			Progress:    t.Progress,
			Format:      t.Format,
			ResultURL:   t.ResultURL,
			Error:       t.Error,
			CreatedAt:   t.CreatedAt,
			RecordCount: t.RecordCount,
		}
		t.mu.Unlock()
	}

	b, _ := json.Marshal(dumps)
	_ = common.DB.Child("network", "site").Put(context.Background(), "export_tasks", string(b), kv.TTLKeep)
}

func (m *ExportManager) saveTasks() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	m.saveTasksLocked()
}

func (m *ExportManager) GetTask(id string) *ExportTaskDTO {
	m.mu.RLock()
	t := m.tasks[id]
	m.mu.RUnlock()

	if t == nil {
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
	m.mu.RLock()
	defer m.mu.RUnlock()
	res := make([]ExportTaskDTO, 0, len(m.tasks))
	for _, t := range m.tasks {
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

func (m *ExportManager) StartCleanupTimer() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			m.Cleanup()
		}
	}()
}

func (m *ExportManager) Cleanup() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	changed := false
	for id, t := range m.tasks {
		t.mu.Lock()
		status := t.Status
		createdAt := t.CreatedAt
		t.mu.Unlock()

		if now.Sub(createdAt) > 24*time.Hour && status != "Running" {
			m.deleteTask(id)
			changed = true
		}
	}
	if changed {
		m.saveTasksLocked()
	}
}

func (m *ExportManager) DeleteTasksByExportID(exportID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	changed := false
	for id := range m.tasks {
		if strings.HasPrefix(id, exportID+"-") {
			m.deleteTask(id)
			changed = true
		}
	}
	if changed {
		m.saveTasksLocked()
	}
}

func (m *ExportManager) deleteTask(id string) {
	t := m.tasks[id]
	if t == nil {
		return
	}
	tempFileName := fmt.Sprintf("site_export_%s.%s", t.ID, t.Format)
	_ = common.TempDir.Remove(filepath.Join("temp", tempFileName))
	delete(m.tasks, id)
}

func (m *ExportManager) TriggerExport(ctx context.Context, exportID string, format string) (string, error) {
	e, err := repo.GetExport(ctx, exportID)
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	// 冲突检查与僵尸任务探测
	var toCancel []*ExportTask
	for _, t := range m.tasks {
		if strings.HasPrefix(t.ID, exportID+"-") {
			t.mu.Lock()
			if t.Status == "Pending" || t.Status == "Running" {
				lockKey := "action:site_export:" + t.ID
				if release := common.Locker.TryLock(ctx, lockKey); release != nil {
					// 僵死任务，标记为 Cancelled 并继续
					t.Status = "Cancelled"
					release()
				} else {
					t.mu.Unlock()
					m.mu.Unlock()
					return "", fmt.Errorf("an export task for %s is already in progress", exportID)
				}
			}
			t.mu.Unlock()
			toCancel = append(toCancel, t)
		}
	}

	taskID := fmt.Sprintf("%s-%d", exportID, time.Now().UnixNano())
	task := &ExportTask{ID: taskID, Status: "Pending", Format: format, CreatedAt: time.Now()}
	m.tasks[taskID] = task
	m.saveTasksLocked()
	m.mu.Unlock()

	m.wg.Add(1)
	go m.runExport(context.Background(), task, e)
	return taskID, nil
}

func (m *ExportManager) WaitAll() {
	m.wg.Wait()
}

func (m *ExportManager) runExport(ctx context.Context, task *ExportTask, e *models.SiteExport) {
	defer m.wg.Done()

	// 获取分布式锁，证明该节点正在处理该任务
	lockKey := "action:site_export:" + task.ID
	release := common.Locker.TryLock(ctx, lockKey)
	if release == nil {
		log.Printf("Site Export %s already handled by another node", task.ID)
		return
	}
	defer release()

	task.mu.Lock()
	if task.Status == "Cancelled" {
		task.mu.Unlock()
		m.saveTasks()
		return
	}
	task.Status = "Running"
	task.mu.Unlock()

	program, err := expr.Compile(e.Rule, expr.Env(map[string]interface{}{
		"tags":   []string{},
		"domain": "",
		"type":   uint8(0),
	}))
	if err != nil {
		updateTaskError(task, "Compile error: "+err.Error())
		m.saveTasks()
		return
	}

	totalEntries := int64(0)
	for _, gid := range e.GroupIDs {
		g, _ := repo.GetGroup(ctx, gid)
		if g != nil {
			totalEntries += g.EntryCount
		}
	}
	if totalEntries == 0 {
		totalEntries = 1
	}

	_ = common.TempDir.MkdirAll("temp", 0755)
	tempFileName := fmt.Sprintf("site_export_%s.%s", task.ID, task.Format)
	tempPath := filepath.Join("temp", tempFileName)
	f, err := common.TempDir.Create(tempPath)
	if err != nil {
		updateTaskError(task, "File create error: "+err.Error())
		m.saveTasks()
		return
	}
	defer f.Close()

	if task.Format == "json" {
		f.WriteString("[\n")
	}
	firstItem := true
	totalRead := int64(0)

	// Subdomain Deduplication Logic (Simple version for MVP)
	// In a real scenario, we would use a Trie to deduplicate across all pools.

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
			task.mu.Lock()
			if task.Status == "Cancelled" {
				task.mu.Unlock()
				pf.Close()
				m.saveTasks()
				return
			}
			task.mu.Unlock()

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
			}
		}
		pf.Close()
	}

	if task.Format == "json" {
		f.WriteString("\n]\n")
	}

	task.mu.Lock()
	if task.Status != "Cancelled" {
		task.Status = "Success"
		task.Progress = 1.0
		task.ResultURL = "/api/v1/network/site/exports/download/" + task.ID
	}
	task.mu.Unlock()
	m.saveTasks()
}

func updateTaskError(t *ExportTask, msg string) {
	t.mu.Lock()
	t.Status = "Failed"
	t.Error = msg
	t.mu.Unlock()
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
