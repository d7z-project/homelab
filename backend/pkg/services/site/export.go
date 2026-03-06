package site

import (
	"context"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
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
	ID        string
	Status    string
	Progress  float64
	Format    string
	ResultURL string
	Error     string
	mu        sync.Mutex
}

type ExportManager struct {
	mu       sync.RWMutex
	tasks    map[string]*ExportTask
	analysis *AnalysisEngine
}

func NewExportManager(analysis *AnalysisEngine) *ExportManager {
	return &ExportManager{
		tasks:    make(map[string]*ExportTask),
		analysis: analysis,
	}
}

func (m *ExportManager) GetTask(id string) *ExportTask {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tasks[id]
}

func (m *ExportManager) TriggerExport(ctx context.Context, exportID string, format string) (string, error) {
	e, err := repo.GetExport(ctx, exportID)
	if err != nil { return "", err }

	m.mu.Lock()
	for id, t := range m.tasks {
		if strings.HasPrefix(id, exportID+":") && t.Status == "Running" {
			t.mu.Lock()
			t.Status = "Cancelled"
			t.mu.Unlock()
		}
	}
	taskID := fmt.Sprintf("%s:%d", exportID, time.Now().UnixNano())
	task := &ExportTask{ID: taskID, Status: "Pending", Format: format}
	m.tasks[taskID] = task
	m.mu.Unlock()

	go m.runExport(context.Background(), task, e)
	return taskID, nil
}

func (m *ExportManager) runExport(ctx context.Context, task *ExportTask, e *models.SiteExport) {
	task.mu.Lock()
	task.Status = "Running"
	task.mu.Unlock()

	program, err := expr.Compile(e.Rule, expr.Env(map[string]interface{}{
		"tags": []string{},
		"domain": "",
		"type": uint8(0),
	}))
	if err != nil {
		updateTaskError(task, "Compile error: "+err.Error())
		return
	}

	totalEntries := int64(0)
	for _, gid := range e.GroupIDs {
		g, _ := repo.GetGroup(ctx, gid)
		if g != nil { totalEntries += g.EntryCount }
	}
	if totalEntries == 0 { totalEntries = 1 }

	_ = common.TempDir.MkdirAll("temp", 0755)
	tempFileName := fmt.Sprintf("site_export_%s.%s", task.ID, task.Format)
	tempPath := filepath.Join("temp", tempFileName)
	f, err := common.TempDir.Create(tempPath)
	if err != nil {
		updateTaskError(task, "File create error: "+err.Error())
		return
	}
	defer f.Close()

	if task.Format == "json" { f.WriteString("[\n") }
	firstItem := true
	processed := int64(0)

	// Subdomain Deduplication Logic (Simple version for MVP)
	// In a real scenario, we would use a Trie to deduplicate across all pools.
	
	for _, gid := range e.GroupIDs {
		poolPath := filepath.Join(PoolsDir, gid+".bin")
		pf, err := common.FS.Open(poolPath)
		if err != nil { continue }
		reader, _ := NewReader(pf)
		for {
			task.mu.Lock()
			if task.Status == "Cancelled" { task.mu.Unlock(); pf.Close(); return }
			task.mu.Unlock()

			entry, err := reader.Next()
			if err == io.EOF { break }
			processed++

			out, err := expr.Run(program, map[string]interface{}{
				"tags": entry.Tags,
				"domain": entry.Value,
				"type": entry.Type,
			})

			if err == nil && out == true {
				if !firstItem && task.Format == "json" { f.WriteString(",\n") }
				writeEntry(f, task.Format, entry)
				firstItem = false
			}

			if processed%1000 == 0 {
				task.mu.Lock()
				task.Progress = float64(processed) / float64(totalEntries)
				task.mu.Unlock()
			}
		}
		pf.Close()
	}

	if task.Format == "json" { f.WriteString("\n]") }

	task.mu.Lock()
	if task.Status != "Cancelled" {
		task.Status = "Success"
		task.Progress = 1.0
		task.ResultURL = "/api/v1/network/site/exports/download/" + task.ID
	}
	task.mu.Unlock()
}

func updateTaskError(t *ExportTask, msg string) {
	t.mu.Lock()
	t.Status = "Failed"
	t.Error = msg
	t.mu.Unlock()
}

func writeEntry(w io.Writer, format string, e models.SitePoolEntry) {
	switch format {
	case "text":
		prefix := ""
		switch e.Type {
		case 0: prefix = "keyword:"
		case 1: prefix = "regexp:"
		case 2: prefix = "domain:"
		case 3: prefix = "full:"
		}
		fmt.Fprintf(w, "%s%s\n", prefix, e.Value)
	case "json":
		b, _ := json.Marshal(e)
		w.Write(b)
	case "yaml":
		fmt.Fprintf(w, "- type: %d\n  value: %s\n", e.Type, e.Value)
		if len(e.Tags) > 0 {
			fmt.Fprintln(w, "  tags:")
			for _, t := range e.Tags { fmt.Fprintf(w, "    - %s\n", t) }
		}
	}
}
