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
	rbacmodel "homelab/pkg/models/core/rbac"
	sitemodel "homelab/pkg/models/network/site"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/site"
	ruleservice "homelab/pkg/services/rules"
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
	Status    shared.TaskStatus `json:"status"`
	Progress  float64           `json:"progress"`
	Error     string            `json:"error"`
	CreatedAt time.Time         `json:"createdAt"`
	mu        sync.Mutex
}

func (t *SyncTask) GetID() string { return t.ID }
func (t *SyncTask) GetStatus() shared.TaskStatus {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.Status
}
func (t *SyncTask) SetStatus(status shared.TaskStatus) {
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

func validateSourceURL(url string, policy *sitemodel.SiteSyncPolicy) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("only http/https URLs are allowed")
	}
	return nil
}

func (s *SitePoolService) CreateSyncPolicy(ctx context.Context, policy *sitemodel.SiteSyncPolicy) error {
	if err := requireSiteResource(ctx, siteResourceBase); err != nil {
		return err
	}

	if policy.ID != "" && !strings.HasPrefix(policy.ID, "sync_") {
		return fmt.Errorf("%w: id for sync policy must start with 'sync_'", common.ErrBadRequest)
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

	err := ruleservice.CreateAndLoad(ctx, repo.SyncPolicyRepo, policy, func(res *shared.Resource[sitemodel.SiteSyncPolicyV1Meta, sitemodel.SiteSyncPolicyV1Status]) error {
		res.Meta = policy.Meta
		res.Status.CreatedAt = time.Now()
		res.Status.UpdatedAt = time.Now()
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	})
	if err == nil && policy.Meta.Enabled {
		s.addCronJob(*policy)
		common.NotifyCluster(ctx, common.EventSiteSyncPolicyChanged, policy.ID)
	}
	commonaudit.FromContext(ctx).Log("CreateSiteSyncPolicy", policy.Meta.Name, "Created", err == nil)
	return err
}

func (s *SitePoolService) UpdateSyncPolicy(ctx context.Context, policy *sitemodel.SiteSyncPolicy) error {
	if err := requireSiteResource(ctx, siteResourceBase); err != nil {
		return err
	}

	err := ruleservice.ReplaceMeta(ctx, repo.SyncPolicyRepo, policy)

	if err == nil {
		updated, _ := repo.GetSyncPolicy(ctx, policy.ID)
		if updated != nil {
			if updated.Meta.Enabled {
				s.addCronJob(*updated)
			} else {
				s.removeCronJob(updated.ID)
			}
		}
		common.NotifyCluster(ctx, common.EventSiteSyncPolicyChanged, policy.ID)
	}
	commonaudit.FromContext(ctx).Log("UpdateSiteSyncPolicy", policy.Meta.Name, "Updated", err == nil)
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
	commonaudit.FromContext(ctx).Log("DeleteSiteSyncPolicy", old.Meta.Name, "Deleted", err == nil)
	return err
}

func (s *SitePoolService) GetSyncPolicy(ctx context.Context, id string) (*sitemodel.SiteSyncPolicy, error) {
	return repo.GetSyncPolicy(ctx, id)
}

func (s *SitePoolService) ScanSyncPolicies(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[sitemodel.SiteSyncPolicy], error) {
	res, err := repo.ScanSyncPolicies(ctx, cursor, limit, search)
	if err != nil {
		return nil, err
	}

	for i := range res.Items {
		if t, ok := s.syncTasks.GetTask(res.Items[i].ID); ok {
			status := t.GetStatus()
			if status == shared.TaskStatusRunning || status == shared.TaskStatusPending {
				res.Items[i].Status.LastStatus = status
				res.Items[i].Status.ErrorMessage = t.Error
				res.Items[i].Status.Progress = t.GetProgress()
			}
		}
	}
	return res, nil
}

func (s *SitePoolService) Sync(ctx context.Context, id string) error {
	if err := requireSiteResource(ctx, siteResourceBase); err != nil {
		return err
	}
	policy, err := repo.GetSyncPolicy(ctx, id)
	if err != nil {
		return err
	}

	existingTask, ok := s.syncTasks.GetTask(id)
	if ok {
		status := existingTask.GetStatus()
		if status == shared.TaskStatusPending || status == shared.TaskStatusRunning {
			lockKey := "action:site_sync:" + id
			if release := common.Locker.TryLock(ctx, lockKey); release != nil {
				s.syncTasks.CancelTask(id)
				release()
			} else {
				return fmt.Errorf("sync is already in progress for policy: %s", policy.Meta.Name)
			}
		}
	}

	task := &SyncTask{
		ID:        id,
		Status:    shared.TaskStatusPending,
		CreatedAt: time.Now(),
	}
	s.syncTasks.AddTask(task)

	policy.Status.LastStatus = shared.TaskStatusPending
	policy.Status.LastRunAt = time.Now()
	_ = repo.SaveSyncPolicy(ctx, policy)

	if common.Subscriber != nil {
		common.NotifyCluster(ctx, common.EventSiteSyncRun, id)
	} else {
		go func() {
			sysCtx := commonauth.WithAuth(context.Background(), &commonauth.AuthContext{
				Type: "sa",
				ID:   "system",
			})
			sysCtx = commonauth.WithPermissions(sysCtx, &rbacmodel.ResourcePermissions{AllowedAll: true})
			_ = s.doSync(sysCtx, id)
		}()
	}

	commonaudit.FromContext(ctx).Log("TriggerSiteSync", policy.Meta.Name, "Triggered Asynchronously", true)
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

		policy.Status.LastStatus = shared.TaskStatusRunning
		_ = repo.SaveSyncPolicy(taskCtx, policy)

		defer func() {
			if policy == nil {
				return
			}
			if errors.Is(taskCtx.Err(), context.Canceled) {
				policy.Status.LastStatus = shared.TaskStatusCancelled
				policy.Status.ErrorMessage = "Task cancelled manually"
			} else if finalErr != nil {
				policy.Status.LastStatus = shared.TaskStatusFailed
				policy.Status.ErrorMessage = finalErr.Error()
			} else {
				policy.Status.LastStatus = shared.TaskStatusSuccess
				policy.Status.ErrorMessage = ""
			}
			policy.Status.LastRunAt = time.Now()
			_ = repo.SaveSyncPolicy(context.Background(), policy)
			commonaudit.FromContext(taskCtx).Log("SiteSyncExecute", policy.Meta.Name, "Finished", finalErr == nil)
		}()

		if policy.Meta.Config == nil {
			policy.Meta.Config = make(map[string]string)
		}

		err = validateSourceURL(policy.Meta.SourceURL, policy)
		if err != nil {
			finalErr = err
			return err
		}

		req, err := http.NewRequestWithContext(taskCtx, "GET", policy.Meta.SourceURL, nil)
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
		if mStr := policy.Meta.Config["tagMapping"]; mStr != "" {
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

		if policy.Meta.Format == "geosite" || policy.Meta.Format == "v2ray-dat" {
			targetCategory := policy.Meta.Config["category"]
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

			additionalTags := strings.Split(policy.Meta.Config["tags"], ",")
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
		release, err = s.lockPool(taskCtx, policy.Meta.TargetGroupID)
		if err != nil {
			finalErr = err
			return err
		}
		defer release()

		var group *sitemodel.SiteGroup
		group, err = repo.GetGroup(taskCtx, policy.Meta.TargetGroupID)
		if err != nil {
			finalErr = err
			return err
		}

		_ = common.FS.MkdirAll(PoolsDir, 0755)
		poolPath := filepath.Join(PoolsDir, policy.Meta.TargetGroupID+".bin")
		tempFile := filepath.Join(PoolsDir, policy.Meta.TargetGroupID+".bin.tmp")

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

					if isFromThisPolicy && policy.Meta.Mode == "overwrite" {
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
		checksum := hex.EncodeToString(hf.Sum(nil))
		count := int64(len(finalEntries))

		if err := common.FS.Rename(tempFile, poolPath); err != nil {
			finalErr = err
			return err
		}

		err = repo.GroupRepo.UpdateStatus(taskCtx, group.ID, func(status *sitemodel.SiteGroupV1Status) {
			status.EntryCount = count
			status.UpdatedAt = time.Now()
			status.Checksum = checksum
		})
		if err == nil {
			notifySitePoolChanged(taskCtx, group.ID)
		}
		finalErr = err
		return err
	})
	return finalErr
}
