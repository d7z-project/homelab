package ip

import (
	"context"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
	"io"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/expr-lang/expr"
)

type ExportTask struct {
	ID        string
	Status    string // Pending, Running, Success, Failed
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
	if err != nil {
		return "", err
	}

	m.mu.Lock()
	// 取消该导出配置之前的任务 (IP.md: Cancel & Replace)
	for id, t := range m.tasks {
		if strings.HasPrefix(id, exportID+":") && t.Status == "Running" {
			t.mu.Lock()
			t.Status = "Cancelled"
			t.mu.Unlock()
		}
	}

	taskID := fmt.Sprintf("%s:%d", exportID, time.Now().UnixNano())
	task := &ExportTask{
		ID:     taskID,
		Status: "Pending",
		Format: format,
	}
	m.tasks[taskID] = task
	m.mu.Unlock()

	go m.runExport(context.Background(), task, e)

	return taskID, nil
}

func (m *ExportManager) runExport(ctx context.Context, task *ExportTask, e *models.IPExport) {
	task.mu.Lock()
	task.Status = "Running"
	task.mu.Unlock()

	// 1. 编译表达式
	// 支持 AST 预分析降级 (Task 9) - 简化版：这里如果只是 Tag in [...] 可以走捷径，
	// 但 expr 内部也已足够快。
	program, err := expr.Compile(e.Rule, expr.Env(map[string]interface{}{
		"tags": []string{},
		"cidr": "",
		"ip":   "",
	}))
	if err != nil {
		task.mu.Lock()
		task.Status = "Failed"
		task.Error = "Compile error: " + err.Error()
		task.mu.Unlock()
		return
	}

	// 2. 遍历依赖的池
	totalEntries := int64(0)
	for _, gid := range e.GroupIDs {
		g, _ := repo.GetGroup(ctx, gid)
		if g != nil {
			totalEntries += g.EntryCount
		}
	}
	if totalEntries == 0 {
		totalEntries = 1 // 避免除零
	}

	processed := int64(0)
	
	tempFileName := fmt.Sprintf("export_%s.%s", task.ID, task.Format)
	tempPath := filepath.Join("temp", tempFileName)
	_ = common.TempDir.MkdirAll("temp", 0755)
	f, err := common.TempDir.Create(tempPath)
	if err != nil {
		task.mu.Lock()
		task.Status = "Failed"
		task.Error = "File create error: " + err.Error()
		task.mu.Unlock()
		return
	}
	defer f.Close()

	if task.Format == "json" {
		f.WriteString("[\n")
	}

	firstJsonItem := true

	for _, gid := range e.GroupIDs {
		poolPath := filepath.Join(PoolsDir, gid+".bin")
		pf, err := common.FS.Open(poolPath)
		if err != nil {
			continue
		}
		reader, _ := NewReader(pf)
		for {
			task.mu.Lock()
			if task.Status == "Cancelled" {
				task.mu.Unlock()
				pf.Close()
				return
			}
			task.mu.Unlock()

			prefix, tags, err := reader.Next()
			if err == io.EOF {
				break
			}
			processed++
			
			// 运行表达式
			output, err := expr.Run(program, map[string]interface{}{
				"tags": tags,
				"cidr": prefix.String(),
				"ip":   prefix.Addr().String(),
			})
			
			if err == nil && output == true {
				// 匹配成功，写入结果
				switch task.Format {
				case "text":
					f.WriteString(prefix.String() + "\n")
				case "json":
					item := map[string]interface{}{"cidr": prefix.String(), "tags": tags}
					b, _ := json.Marshal(item)
					if !firstJsonItem {
						f.WriteString(",\n")
					}
					f.Write(b)
					firstJsonItem = false
				case "yaml":
					f.WriteString(fmt.Sprintf("- cidr: %s\n", prefix.String()))
					if len(tags) > 0 {
						f.WriteString("  tags:\n")
						for _, t := range tags {
							f.WriteString(fmt.Sprintf("    - %s\n", t))
						}
					}
				}
			}

			if processed%1000 == 0 {
				task.mu.Lock()
				task.Progress = float64(processed) / float64(totalEntries)
				task.mu.Unlock()
			}
		}
		pf.Close()
	}

	if task.Format == "json" {
		f.WriteString("\n]")
	}

	task.mu.Lock()
	if task.Status != "Cancelled" {
		task.Status = "Success"
		task.Progress = 1.0
		task.ResultURL = "/api/v1/network/ip/exports/download/" + task.ID
	}
	task.mu.Unlock()
}
