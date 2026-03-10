package site

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	taskpkg "homelab/pkg/common/task"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/site"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/afero"
	"golang.org/x/net/idna"
)

type SyncTask struct {
	ID        string            `json:"id"`
	Status    models.TaskStatus `json:"status"`
	Progress  float64           `json:"progress"`
	Error     string            `json:"error"`
	CreatedAt time.Time         `json:"createdAt"`
	mu        sync.Mutex
}

func (t *SyncTask) GetID() string { return t.ID }
func (t *SyncTask) GetStatus() models.TaskStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Status
}
func (t *SyncTask) SetStatus(status models.TaskStatus) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Status = status
}
func (t *SyncTask) SetError(err string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Error = err
}
func (t *SyncTask) GetCreatedAt() time.Time { return t.CreatedAt }
func (t *SyncTask) GetProgress() float64 {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Progress
}
func (t *SyncTask) SetProgress(p float64) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.Progress = p
}

func validateSourceURL(url string, policy *models.SiteSyncPolicy) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("only http/https URLs are allowed")
	}
	return nil
}

func (s *SitePoolService) CreateSyncPolicy(ctx context.Context, policy *models.SiteSyncPolicy) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/site") {
		return fmt.Errorf("%w: network/site", commonauth.ErrPermissionDenied)
	}
	if policy.ID == "" {
		for i := 0; i < 10; i++ {
			newID := generatePolicyID()
			if _, err := repo.GetSyncPolicy(ctx, newID); err != nil {
				policy.ID = newID
				break
			}
		}
		if policy.ID == "" {
			return fmt.Errorf("failed to generate unique policy ID")
		}
	}
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	err := repo.SaveSyncPolicy(ctx, policy)
	if err == nil && policy.Enabled {
		s.addCronJob(*policy)
		common.NotifyCluster(ctx, common.EventSiteSyncPolicyChanged, policy.ID)
	}
	commonaudit.FromContext(ctx).Log("CreateSiteSyncPolicy", policy.Name, "Created", err == nil)
	return err
}

func (s *SitePoolService) UpdateSyncPolicy(ctx context.Context, policy *models.SiteSyncPolicy) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/site") {
		return fmt.Errorf("%w: network/site", commonauth.ErrPermissionDenied)
	}
	old, err := repo.GetSyncPolicy(ctx, policy.ID)
	if err != nil {
		return err
	}
	policy.CreatedAt = old.CreatedAt
	policy.UpdatedAt = time.Now()
	err = repo.SaveSyncPolicy(ctx, policy)
	if err == nil {
		if policy.Enabled {
			s.addCronJob(*policy)
		} else {
			s.removeCronJob(policy.ID)
		}
		common.NotifyCluster(ctx, common.EventSiteSyncPolicyChanged, policy.ID)
	}
	commonaudit.FromContext(ctx).Log("UpdateSiteSyncPolicy", policy.Name, "Updated", err == nil)
	return err
}

func (s *SitePoolService) DeleteSyncPolicy(ctx context.Context, id string) error {
	old, err := repo.GetSyncPolicy(ctx, id)
	if err != nil {
		return err
	}
	err = repo.DeleteSyncPolicy(ctx, id)
	if err == nil {
		s.removeCronJob(id)
		common.NotifyCluster(ctx, common.EventSiteSyncPolicyChanged, id)
	}
	commonaudit.FromContext(ctx).Log("DeleteSiteSyncPolicy", old.Name, "Deleted", err == nil)
	return err
}

func (s *SitePoolService) GetSyncPolicy(ctx context.Context, id string) (*models.SiteSyncPolicy, error) {
	return repo.GetSyncPolicy(ctx, id)
}

func (s *SitePoolService) ScanSyncPolicies(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.SiteSyncPolicy], error) {
	res, err := repo.ScanSyncPolicies(ctx, cursor, limit, search)
	if err != nil {
		return nil, err
	}

	for i := range res.Items {
		if t, ok := s.syncTasks.GetTask(res.Items[i].ID); ok {
			status := t.GetStatus()
			if status == models.TaskStatusRunning || status == models.TaskStatusPending {
				res.Items[i].LastStatus = status
				res.Items[i].ErrorMessage = t.Error
				res.Items[i].Progress = t.GetProgress()
			}
		}
	}
	return res, nil
}

func (s *SitePoolService) Sync(ctx context.Context, id string) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/site") {
		return fmt.Errorf("%w: network/site", commonauth.ErrPermissionDenied)
	}
	policy, err := repo.GetSyncPolicy(ctx, id)
	if err != nil {
		return err
	}

	existingTask, ok := s.syncTasks.GetTask(id)
	if ok {
		status := existingTask.GetStatus()
		if status == models.TaskStatusPending || status == models.TaskStatusRunning {
			lockKey := "action:site_sync:" + id
			if release := common.Locker.TryLock(ctx, lockKey); release != nil {
				s.syncTasks.CancelTask(id)
				release()
			} else {
				return fmt.Errorf("sync is already in progress for policy: %s", policy.Name)
			}
		}
	}

	task := &SyncTask{
		ID:        id,
		Status:    models.TaskStatusPending,
		CreatedAt: time.Now(),
	}
	s.syncTasks.AddTask(task)

	policy.LastStatus = models.TaskStatusPending
	policy.LastRunAt = time.Now()
	_ = repo.SaveSyncPolicy(ctx, policy)

	if common.Subscriber != nil {
		common.NotifyCluster(ctx, common.EventSiteSyncRun, id)
	} else {
		go func() {
			sysCtx := commonauth.WithAuth(context.Background(), &commonauth.AuthContext{
				Type: "sa",
				ID:   "system",
			})
			sysCtx = commonauth.WithPermissions(sysCtx, &models.ResourcePermissions{AllowedAll: true})
			_ = s.doSync(sysCtx, id)
		}()
	}

	commonaudit.FromContext(ctx).Log("TriggerSiteSync", policy.Name, "Triggered Asynchronously", true)
	return nil
}

type syncAggKey struct {
	Type  uint8
	Value string
}

func (s *SitePoolService) doSync(bgCtx context.Context, policyID string) error {
	var finalErr error
	s.syncTasks.RunTask(bgCtx, policyID, func(taskCtx context.Context, task *SyncTask) error {
		policy, err := repo.GetSyncPolicy(taskCtx, policyID)
		if err != nil || policy == nil {
			finalErr = fmt.Errorf("policy is nil or missing")
			return finalErr
		}

		policy.LastStatus = models.TaskStatusRunning
		_ = repo.SaveSyncPolicy(taskCtx, policy)

		defer func() {
			if policy == nil {
				return
			}
			if errors.Is(taskCtx.Err(), context.Canceled) {
				policy.LastStatus = models.TaskStatusCancelled
				policy.ErrorMessage = "Task cancelled manually"
			} else if finalErr != nil {
				policy.LastStatus = models.TaskStatusFailed
				policy.ErrorMessage = finalErr.Error()
			} else {
				policy.LastStatus = models.TaskStatusSuccess
				policy.ErrorMessage = ""
			}
			policy.LastRunAt = time.Now()
			_ = repo.SaveSyncPolicy(context.Background(), policy)
			commonaudit.FromContext(taskCtx).Log("SiteSyncExecute", policy.Name, "Finished", finalErr == nil)
		}()

		if policy.Config == nil {
			policy.Config = make(map[string]string)
		}

		err = validateSourceURL(policy.SourceURL, policy)
		if err != nil {
			finalErr = err
			return err
		}

		req, err := http.NewRequestWithContext(taskCtx, "GET", policy.SourceURL, nil)
		if err != nil {
			finalErr = err
			return err
		}

		client := &http.Client{Timeout: 300 * time.Second, Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		}}
		resp, err := client.Do(req)
		if err != nil {
			finalErr = err
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			finalErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			return finalErr
		}

		tempSrc, err := os.CreateTemp("", "sync_site_src_*.tmp")
		if err != nil {
			finalErr = err
			return err
		}
		defer os.Remove(tempSrc.Name())
		defer tempSrc.Close()

		reader := taskpkg.NewProgressReader[*SyncTask](resp.Body, resp.ContentLength, task, s.syncTasks)

		if _, err := io.Copy(tempSrc, reader); err != nil {
			finalErr = err
			return err
		}
		_ = tempSrc.Close()

		newTags := make([]string, 0)
		tagMap := make(map[string]uint32)

		tagMapping := make(map[string]string)
		if mStr := policy.Config["tagMapping"]; mStr != "" {
			_ = json.Unmarshal([]byte(mStr), &tagMapping)
		}

		getTagIdx := func(t string) uint32 {
			t = strings.ToLower(strings.TrimSpace(t))
			if mapped, ok := tagMapping[t]; ok {
				t = strings.ToLower(mapped)
			}
			if idx, ok := tagMap[t]; ok {
				return idx
			}
			idx := uint32(len(newTags))
			newTags = append(newTags, t)
			tagMap[t] = idx
			return idx
		}

		internalTagIdx := getTagIdx(policy.ID)
		aggregate := make(map[syncAggKey]map[uint32]struct{})

		addEntryToAggregate := func(entryType uint8, val string, tagIdxs []uint32) {
			key := syncAggKey{Type: entryType, Value: val}
			if _, ok := aggregate[key]; !ok {
				aggregate[key] = make(map[uint32]struct{})
			}
			for _, idx := range tagIdxs {
				aggregate[key][idx] = struct{}{}
			}
		}

		if policy.Format == "geosite" || policy.Format == "v2ray-dat" {
			targetCategory := policy.Config["category"]
			importAll := targetCategory == "" || targetCategory == "*" || targetCategory == "all"

			var data []byte
			data, err = os.ReadFile(tempSrc.Name())
			if err != nil {
				return err
			}

			var v2Entries []ParsedGeoSiteEntry
			v2Entries, err = ParseV2RayGeoSite(data, targetCategory, importAll)
			if err != nil {
				return err
			}

			for _, e := range v2Entries {
				val, err := idna.ToASCII(strings.ToLower(e.Value))
				if err != nil {
					val = strings.ToLower(e.Value)
				}
				addEntryToAggregate(e.Type, val, []uint32{internalTagIdx, getTagIdx(e.Category)})
			}
		} else {
			var data []byte
			data, err = os.ReadFile(tempSrc.Name())
			if err != nil {
				return err
			}

			additionalTags := strings.Split(policy.Config["tags"], ",")
			var globalTagIdxs []uint32
			for _, t := range additionalTags {
				t = strings.TrimSpace(t)
				if t != "" {
					globalTagIdxs = append(globalTagIdxs, getTagIdx(t))
				}
			}
			if len(globalTagIdxs) == 0 {
				globalTagIdxs = append(globalTagIdxs, getTagIdx("sync"))
			}

			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				select {
				case <-taskCtx.Done():
					finalErr = context.Canceled
					return finalErr
				default:
				}

				if common.IsComment(line) {
					continue
				}

				eType := uint8(2) // Default domain
				val := line
				if strings.HasPrefix(line, "full:") {
					eType = 3
					val = strings.TrimPrefix(line, "full:")
				} else if strings.HasPrefix(line, "domain:") {
					eType = 2
					val = strings.TrimPrefix(line, "domain:")
				} else if strings.HasPrefix(line, "keyword:") {
					eType = 0
					val = strings.TrimPrefix(line, "keyword:")
				} else if strings.HasPrefix(line, "regexp:") {
					eType = 1
					val = strings.TrimPrefix(line, "regexp:")
				}

				normVal, err := idna.ToASCII(strings.ToLower(val))
				if err == nil {
					val = normVal
				}

				newIdxs := append([]uint32{internalTagIdx}, globalTagIdxs...)
				addEntryToAggregate(eType, val, newIdxs)
			}
		}

		if len(aggregate) == 0 {
			finalErr = fmt.Errorf("no valid domain entries found in source")
			return finalErr
		}

		var release func()
		release, err = s.lockPool(taskCtx, policy.TargetGroupID)
		if err != nil {
			finalErr = err
			return err
		}
		defer release()

		var group *models.SiteGroup
		group, err = repo.GetGroup(taskCtx, policy.TargetGroupID)
		if err != nil {
			finalErr = err
			return err
		}

		_ = common.FS.MkdirAll(PoolsDir, 0755)
		poolPath := filepath.Join(PoolsDir, policy.TargetGroupID+".bin")
		tempFile := filepath.Join(PoolsDir, policy.TargetGroupID+".bin.tmp")

		if exists, _ := afero.Exists(common.FS, poolPath); exists {
			var pf afero.File
			pf, err = common.FS.Open(poolPath)
			if err == nil {
				reader, _ := NewReader(pf)
				targetInternalTag := strings.ToLower(policy.ID)

				for {
					entry, err := reader.Next()
					if err == io.EOF {
						break
					}

					isFromThisPolicy := false
					var otherTags []string
					hasOtherPolicy := false

					for _, tagName := range entry.Tags {
						if tagName == targetInternalTag {
							isFromThisPolicy = true
						} else {
							if strings.HasPrefix(tagName, "_") {
								hasOtherPolicy = true
							}
							otherTags = append(otherTags, tagName)
						}
					}

					if isFromThisPolicy && policy.Mode == "overwrite" {
						if !hasOtherPolicy {
							continue
						} else {
							var remainingIdxs []uint32
							for _, ot := range otherTags {
								remainingIdxs = append(remainingIdxs, getTagIdx(ot))
							}
							addEntryToAggregate(entry.Type, entry.Value, remainingIdxs)
							continue
						}
					}

					var convertedIdxs []uint32
					for _, t := range entry.Tags {
						convertedIdxs = append(convertedIdxs, getTagIdx(t))
					}
					addEntryToAggregate(entry.Type, entry.Value, convertedIdxs)
				}
				pf.Close()
			}
		}

		var finalEntries []Entry
		for key, tagSet := range aggregate {
			var idxs []uint32
			for idx := range tagSet {
				idxs = append(idxs, idx)
			}
			finalEntries = append(finalEntries, Entry{Type: key.Type, Value: key.Value, TagIndices: idxs})
		}

		finalTags := newTags
		var tf afero.File
		tf, err = common.FS.Create(tempFile)
		if err != nil {
			finalErr = err
			return err
		}
		codec := NewCodec()
		err = codec.WritePool(tf, finalTags, finalEntries)
		tf.Close()
		if err != nil {
			_ = common.FS.Remove(tempFile)
			finalErr = err
			return err
		}

		var content []byte
		content, err = afero.ReadFile(common.FS, tempFile)
		if err != nil {
			_ = common.FS.Remove(tempFile)
			finalErr = err
			return err
		}
		hf := sha256.New()
		hf.Write(content)

		group.EntryCount = int64(len(finalEntries))
		group.UpdatedAt = time.Now()
		group.Checksum = hex.EncodeToString(hf.Sum(nil))

		if err := common.FS.Rename(tempFile, poolPath); err != nil {
			finalErr = err
			return err
		}

		err = repo.SaveGroup(taskCtx, group)
		if err == nil {
			notifySitePoolChanged(taskCtx, group.ID)
		}
		finalErr = err
		return err
	})
	return finalErr
}
