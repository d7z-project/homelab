package ip

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/common/task"
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
)

type ExportTask struct {
	ID          string            `json:"id"`
	Status      models.TaskStatus `json:"status"` // Pending, Running, Success, Failed, Cancelled
	Progress    float64           `json:"progress"`
	Format      string            `json:"format"`
	ResultURL   string            `json:"resultUrl"`
	Error       string            `json:"error"`
	CreatedAt   time.Time         `json:"createdAt"`
	RecordCount int64             `json:"recordCount"`
	Checksum    string            `json:"checksum"` // Rule + GroupChecksums + Format
	mu          sync.Mutex
}

func (t *ExportTask) GetID() string { return t.ID }
func (t *ExportTask) GetStatus() models.TaskStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Status
}
func (t *ExportTask) SetStatus(status models.TaskStatus) {
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
func (t *ExportTask) GetProgress() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Progress
}
func (t *ExportTask) SetProgress(p float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Progress = p
}

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
		Checksum:    t.Checksum,
	})
}

// Ensure interface implementation at compile time
var _ models.TaskInfo = (*ExportTask)(nil)

type ExportTaskDTO struct {
	ID          string            `json:"id"`
	Status      models.TaskStatus `json:"status"`
	Progress    float64           `json:"progress"`
	Format      string            `json:"format"`
	ResultURL   string            `json:"resultUrl"`
	Error       string            `json:"error"`
	CreatedAt   time.Time         `json:"createdAt"`
	RecordCount int64             `json:"recordCount"`
	Checksum    string            `json:"checksum"`
}

type ExportManager struct {
	core     *task.Manager[*ExportTask]
	analysis *AnalysisEngine
	wg       sync.WaitGroup
}

func NewExportManager(analysis *AnalysisEngine) *ExportManager {
	core := task.NewManager[*ExportTask]("action:ip_export", "export_tasks", "network", "ip")

	core.SetCleanupHook(func(t *ExportTask) {
		tempFileName := fmt.Sprintf("export_%s.%s", t.ID, t.Format)
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
		Checksum:    t.Checksum,
	}
}

func (m *ExportManager) ScanTasks() []ExportTaskDTO {
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
			Checksum:    t.Checksum,
		})
		t.mu.Unlock()
	}
	return res
}

func (m *ExportManager) DeleteTasksByExportID(exportID string) {
	m.core.DeleteTasksByPrefix(exportID + "-")
}

func (m *ExportManager) CancelTask(id string) bool {
	return m.core.CancelTask(id)
}

func (m *ExportManager) TriggerExport(ctx context.Context, exportID string, format string) (string, error) {
	e, err := repo.GetExport(ctx, exportID)
	if err != nil {
		return "", err
	}

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

	// Check cache
	tasks := m.core.RangeAll()
	for _, t := range tasks {
		t.mu.Lock()
		match := t.Checksum == currentChecksum && t.Status == models.TaskStatusSuccess
		tID := t.ID
		tFormat := t.Format
		t.mu.Unlock()
		if match {
			tempFileName := fmt.Sprintf("export_%s.%s", tID, tFormat)
			tempPath := filepath.Join("temp", tempFileName)
			if exists, _ := afero.Exists(common.TempDir, tempPath); exists {
				return tID, nil
			}
		}
	}

	// Conflict check / Running
	for _, t := range tasks {
		if strings.HasPrefix(t.ID, exportID+"-") {
			status := t.GetStatus()
			if status == models.TaskStatusPending || status == models.TaskStatusRunning {
				lockKey := "action:ip_export:" + t.ID
				if release := common.Locker.TryLock(ctx, lockKey); release != nil {
					m.core.CancelTask(t.ID)
					release()
				} else {
					return "", fmt.Errorf("an export task for %s is already in progress", exportID)
				}
			}
		}
	}

	taskID := fmt.Sprintf("%s-%d", exportID, time.Now().UnixNano())
	task := &ExportTask{
		ID:        taskID,
		Status:    models.TaskStatusPending,
		Format:    format,
		Checksum:  currentChecksum,
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

func (m *ExportManager) runExport(bgCtx context.Context, taskID string, e *models.IPExport) {
	defer m.wg.Done()

	m.core.RunTask(bgCtx, taskID, func(taskCtx context.Context, task *ExportTask) error {
		program, err := expr.Compile(e.Rule, expr.Env(map[string]interface{}{
			"tags": []string{},
			"cidr": "",
			"ip":   "",
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

		totalRead := int64(0)
		tempFileName := fmt.Sprintf("export_%s.%s", task.ID, task.Format)
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
				select {
				case <-taskCtx.Done():
					pf.Close()
					return context.Canceled
				default:
				}

				prefix, tags, err := reader.Next()
				if err == io.EOF {
					break
				}
				totalRead++

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
					m.core.Save()
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
		task.Progress = 1.0
		task.ResultURL = "/api/v1/network/ip/exports/download/" + task.ID
		task.mu.Unlock()
		return nil
	})
}
