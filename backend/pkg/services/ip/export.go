package ip

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
	"io"
	"net"
	"net/netip"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
	"github.com/spf13/afero"
	"gopkg.d7z.net/middleware/kv"
)

type ExportTask struct {
	ID          string
	Status      string // Pending, Running, Success, Failed
	Progress    float64
	Format      string
	ResultURL   string
	Error       string
	CreatedAt   time.Time
	RecordCount int64
	Checksum    string // Rule + GroupChecksums + Format
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
	Checksum    string    `json:"Checksum"`
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
	data, err := common.DB.Child("network", "ip").Get(context.Background(), "export_tasks")
	if err == nil && data != "" {
		_ = json.Unmarshal([]byte(data), &m.tasks)
		for _, t := range m.tasks {
			if t.Status == "Running" || t.Status == "Pending" {
				t.Status = "Failed"
				t.Error = "Server restarted"
			}
		}
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
			Checksum:    t.Checksum,
		}
		t.mu.Unlock()
	}

	b, _ := json.Marshal(dumps)
	_ = common.DB.Child("network", "ip").Put(context.Background(), "export_tasks", string(b), kv.TTLKeep)
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
		Checksum:    t.Checksum,
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
			Checksum:    t.Checksum,
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

		// 清理超过 24 小时的任务
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
	// 删除物理文件
	tempFileName := fmt.Sprintf("export_%s.%s", t.ID, t.Format)
	tempPath := filepath.Join("temp", tempFileName)
	_ = common.TempDir.Remove(tempPath)
	delete(m.tasks, id)
}
func (m *ExportManager) TriggerExport(ctx context.Context, exportID string, format string) (string, error) {
	e, err := repo.GetExport(ctx, exportID)
	if err != nil {
		return "", err
	}

	// 计算当前导出的 Checksum
	hf := sha256.New()
	hf.Write([]byte(e.Rule))
	hf.Write([]byte(format))
	for _, gid := range e.GroupIDs {
		hf.Write([]byte(gid))
		g, _ := repo.GetGroup(ctx, gid)
		if g != nil {
			hf.Write([]byte(g.Checksum))
		}
	}
	currentChecksum := hex.EncodeToString(hf.Sum(nil))

	m.mu.Lock()
	// 检查缓存 (Task 10)
	for _, t := range m.tasks {
		if t.Checksum == currentChecksum && t.Status == "Success" {
			// 检查物理文件是否真的还在
			tempFileName := fmt.Sprintf("export_%s.%s", t.ID, t.Format)
			tempPath := filepath.Join("temp", tempFileName)
			if exists, _ := afero.Exists(common.TempDir, tempPath); exists {
				m.mu.Unlock()
				return t.ID, nil
			}
		}
	}

	// 取消该导出配置之前的任务
	for id, t := range m.tasks {
		if strings.HasPrefix(id, exportID+"-") {
			t.mu.Lock()
			if t.Status == "Running" || t.Status == "Pending" {
				t.Status = "Cancelled"
			}
			t.mu.Unlock()
		}
	}

	taskID := fmt.Sprintf("%s-%d", exportID, time.Now().UnixNano())
	task := &ExportTask{
		ID:        taskID,
		Status:    "Pending",
		Format:    format,
		Checksum:  currentChecksum,
		CreatedAt: time.Now(),
	}
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

func (m *ExportManager) runExport(ctx context.Context, task *ExportTask, e *models.IPExport) {
	defer m.wg.Done()
	task.mu.Lock()
	if task.Status == "Cancelled" {
		task.mu.Unlock()
		m.saveTasks()
		return
	}
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
		m.saveTasks()
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

	totalRead := int64(0)

	tempFileName := fmt.Sprintf("export_%s.%s", task.ID, task.Format)
	tempPath := filepath.Join("temp", tempFileName)
	_ = common.TempDir.MkdirAll("temp", 0755)
	f, err := common.TempDir.Create(tempPath)
	if err != nil {
		task.mu.Lock()
		task.Status = "Failed"
		task.Error = "File create error: " + err.Error()
		task.mu.Unlock()
		m.saveTasks()
		return
	}
	defer f.Close()

	if task.Format == "json" {
		f.WriteString("[\n")
	}

	firstJsonItem := true

	var mmdbWriter *mmdbwriter.Tree
	var v2rayGroups map[string][]netip.Prefix

	if task.Format == "mmdb" {
		mmdbWriter, _ = mmdbwriter.New(mmdbwriter.Options{
			DatabaseType: "GeoIP2-Country",
			RecordSize:   24,
		})
	} else if task.Format == "v2ray-dat" {
		v2rayGroups = make(map[string][]netip.Prefix)
	}

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

			prefix, tags, err := reader.Next()
			if err == io.EOF {
				break
			}
			totalRead++

			// 运行表达式
			output, err := expr.Run(program, map[string]interface{}{
				"tags": tags,
				"cidr": prefix.String(),
				"ip":   prefix.Addr().String(),
			})

			if err == nil && output == true {
				task.mu.Lock()
				task.RecordCount++
				task.mu.Unlock()

				var publicTags []string
				for _, t := range tags {
					if !strings.HasPrefix(t, "_") {
						publicTags = append(publicTags, t)
					}
				}

				// 匹配成功，写入结果
				switch task.Format {
				case "text":
					f.WriteString(prefix.String() + "\n")
				case "json":
					item := map[string]interface{}{"cidr": prefix.String(), "tags": publicTags}
					b, _ := json.Marshal(item)
					if !firstJsonItem {
						f.WriteString(",\n")
					}
					f.Write(b)
					firstJsonItem = false
				case "yaml":
					f.WriteString(fmt.Sprintf("- cidr: %s\n", prefix.String()))
					if len(publicTags) > 0 {
						f.WriteString("  tags:\n")
						for _, t := range publicTags {
							f.WriteString(fmt.Sprintf("    - %s\n", t))
						}
					}
				case "v2ray-dat":
					if len(publicTags) == 0 {
						v2rayGroups["UNKNOWN"] = append(v2rayGroups["UNKNOWN"], prefix)
					} else {
						for _, t := range publicTags {
							v2rayGroups[strings.ToUpper(t)] = append(v2rayGroups[strings.ToUpper(t)], prefix)
						}
					}
				case "mmdb":
					code := "XX"
					if len(publicTags) > 0 {
						code = publicTags[0]
					}
					record := mmdbtype.Map{
						"country": mmdbtype.Map{
							"iso_code": mmdbtype.String(strings.ToUpper(code)),
						},
					}
					_, importNet, _ := net.ParseCIDR(prefix.String())
					if importNet != nil {
						_ = mmdbWriter.Insert(importNet, record)
					}
				}
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
	} else if task.Format == "v2ray-dat" {
		_ = BuildV2RayGeoIP(f, v2rayGroups)
	} else if task.Format == "mmdb" {
		_, _ = mmdbWriter.WriteTo(f)
	}

	task.mu.Lock()
	if task.Status != "Cancelled" {
		task.Status = "Success"
		task.Progress = 1.0
		task.ResultURL = "/api/v1/network/ip/exports/download/" + task.ID
	}
	task.mu.Unlock()
	m.saveTasks()
}
