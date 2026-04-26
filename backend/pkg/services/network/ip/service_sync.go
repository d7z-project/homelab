package ip

import (
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	taskpkg "homelab/pkg/common/task"
	ipmodel "homelab/pkg/models/network/ip"
	"homelab/pkg/models/shared"
	repo "homelab/pkg/repositories/network/ip"
	"io"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/oschwald/maxminddb-golang/v2"
	"github.com/spf13/afero"
)

func (s *IPPoolService) CreateSyncPolicy(ctx context.Context, policy *ipmodel.IPSyncPolicy) error {
	if err := requireIPResource(ctx, ipResourceBase); err != nil {
		return err
	}

	if policy.ID != "" && !strings.HasPrefix(policy.ID, "sync_") {
		return fmt.Errorf("%w: id for sync policy must start with 'sync_'", common.ErrBadRequest)
	}

	if policy.ID == "" {
		for i := 0; i < 10; i++ { // 最多重试 10 次
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
	policy.Status.CreatedAt = time.Now()
	policy.Status.UpdatedAt = time.Now()
	err := repo.SaveSyncPolicy(ctx, policy)
	if err == nil && policy.Meta.Enabled {
		s.addCronJob(*policy)
		common.NotifyCluster(ctx, common.EventIPSyncPolicyChanged, policy.ID)
	}
	commonaudit.FromContext(ctx).Log("CreateIPSyncPolicy", policy.Meta.Name, "Created", err == nil)
	return err
}

func (s *IPPoolService) UpdateSyncPolicy(ctx context.Context, policy *ipmodel.IPSyncPolicy) error {
	if err := requireIPResource(ctx, ipResourceBase); err != nil {
		return err
	}

	current, err := repo.GetSyncPolicy(ctx, policy.ID)
	if err == nil {
		current.Meta = policy.Meta
		current.Status.UpdatedAt = time.Now()
		err = repo.SaveSyncPolicy(ctx, current)
	}

	if err == nil {
		updated, _ := repo.GetSyncPolicy(ctx, policy.ID)
		if updated != nil {
			if updated.Meta.Enabled {
				s.addCronJob(*updated)
			} else {
				s.removeCronJob(updated.ID)
			}
		}
		common.NotifyCluster(ctx, common.EventIPSyncPolicyChanged, policy.ID)
	}
	commonaudit.FromContext(ctx).Log("UpdateIPSyncPolicy", policy.Meta.Name, "Updated", err == nil)
	return err
}

func (s *IPPoolService) DeleteSyncPolicy(ctx context.Context, id string) error {
	if err := requireIPResource(ctx, ipResourceBase); err != nil {
		return err
	}
	old, err := repo.GetSyncPolicy(ctx, id)
	if err != nil {
		return err
	}
	err = repo.DeleteSyncPolicy(ctx, id)
	if err == nil {
		s.removeCronJob(id)
		common.NotifyCluster(ctx, common.EventIPSyncPolicyChanged, id)
	}
	commonaudit.FromContext(ctx).Log("DeleteIPSyncPolicy", old.Meta.Name, "Deleted", err == nil)
	return err
}

func (s *IPPoolService) GetSyncPolicy(ctx context.Context, id string) (*ipmodel.IPSyncPolicy, error) {
	if err := requireIPResource(ctx, ipResourceBase); err != nil {
		return nil, err
	}
	return repo.GetSyncPolicy(ctx, id)
}

func (s *IPPoolService) ScanSyncPolicies(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[ipmodel.IPSyncPolicy], error) {
	if err := requireIPResource(ctx, ipResourceBase); err != nil {
		return nil, err
	}
	res, err := repo.ScanSyncPolicies(ctx, cursor, limit, search)
	if err != nil {
		return nil, err
	}

	// 从内存 `manager` 获取最新运行状态、错误和进度
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

func (s *IPPoolService) Sync(ctx context.Context, id string) error {
	if err := requireIPResource(ctx, ipResourceBase); err != nil {
		return err
	}
	policy, err := repo.GetSyncPolicy(ctx, id)
	if err != nil {
		return err
	}

	// 冲突校验接管给框架，只检查本地记录是否有正在进行的 Task
	// 因为 task 字典和政策配置是一对一同步的
	existingTask, ok := s.syncTasks.GetTask(id)
	if ok {
		status := existingTask.GetStatus()
		if status == shared.TaskStatusPending || status == shared.TaskStatusRunning {
			lockKey := "action:ip_sync:" + id
			if release := s.deps.Locker.TryLock(ctx, lockKey); release != nil {
				// 任务僵死：框架强制标记取消
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
	policy.Status.Progress = 0
	policy.Status.ErrorMessage = ""
	policy.Status.QueueTopic = ipSyncTopic
	policy.Status.QueueMessageID = ""
	policy.Status.QueuedAt = nil
	policy.Status.DispatchedAt = nil
	_ = repo.SaveSyncPolicy(ctx, policy)

	payload, err := json.Marshal(syncJob{PolicyID: id})
	if err != nil {
		task.SetStatus(shared.TaskStatusFailed)
		task.SetError(err.Error())
		s.syncTasks.Save()
		policy.Status.LastStatus = shared.TaskStatusFailed
		policy.Status.ErrorMessage = err.Error()
		_ = repo.SaveSyncPolicy(ctx, policy)
		return fmt.Errorf("failed to encode sync job: %w", err)
	}
	messageID, err := s.deps.Queue.Publish(ctx, ipSyncTopic, string(payload), nil)
	if err != nil {
		task.SetStatus(shared.TaskStatusFailed)
		task.SetError(err.Error())
		s.syncTasks.Save()
		policy.Status.LastStatus = shared.TaskStatusFailed
		policy.Status.ErrorMessage = err.Error()
		_ = repo.SaveSyncPolicy(ctx, policy)
		return fmt.Errorf("failed to enqueue sync job: %w", err)
	}
	queuedAt := time.Now()
	policy.Status.QueueMessageID = messageID
	policy.Status.QueuedAt = &queuedAt
	_ = repo.SaveSyncPolicy(ctx, policy)

	commonaudit.FromContext(ctx).Log("TriggerIPSync", policy.Meta.Name, "Queued", true)
	return nil
}

func (s *IPPoolService) doSync(bgCtx context.Context, policyID string) error {
	var finalErr error
	s.syncTasks.RunTask(bgCtx, policyID, func(taskCtx context.Context, task *SyncTask) error {
		// 重新加载 policy 以获取最新配置
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
			// 使用闭包捕获的命名返回值以确保最后的保存
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
			commonaudit.FromContext(taskCtx).Log("IPSyncExecute", policy.Meta.Name, "Finished", finalErr == nil)
		}()

		if policy.Meta.Config == nil {
			policy.Meta.Config = make(map[string]string)
		}

		// 1. 下载原始数据到临时文件
		// SSRF 防护：校验 URL
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

		tempSrc, err := os.CreateTemp("", "sync_src_*.tmp")
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

		// 2. 解析数据
		newTags := make([]string, 0)
		tagMap := make(map[string]uint32)

		// 解析标签映射配置
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

		// 使用 Map 聚合 CIDR 和 Tags 以实现去重合并
		aggregate := make(map[netip.Prefix]map[uint32]struct{})

		addEntryToAggregate := func(prefix netip.Prefix, tagIdxs []uint32) {
			if _, ok := aggregate[prefix]; !ok {
				aggregate[prefix] = make(map[uint32]struct{})
			}
			for _, idx := range tagIdxs {
				aggregate[prefix][idx] = struct{}{}
			}
		}

		if policy.Meta.Format == "geoip" {
			var mdb *maxminddb.Reader
			mdb, err = maxminddb.Open(tempSrc.Name())
			if err != nil {
				return fmt.Errorf("failed to open as maxminddb: %w", err)
			}
			defer mdb.Close()

			for result := range mdb.Networks() {
				select {
				case <-taskCtx.Done():
					finalErr = context.Canceled
					return finalErr
				default:
				}

				record := &struct {
					Country struct {
						IsoCode string `maxminddb:"iso_code"`
					} `maxminddb:"country"`
					City struct {
						Names map[string]string `maxminddb:"names"`
					} `maxminddb:"city"`
				}{}
				if err := result.Decode(record); err != nil {
					continue
				}
				prefix := result.Prefix()
				idxs := []uint32{internalTagIdx}
				if record.Country.IsoCode != "" {
					idxs = append(idxs, getTagIdx(record.Country.IsoCode))
				}
				if record.City.Names != nil {
					lang := policy.Meta.Config["language"]
					if lang == "" {
						lang = "zh-CN"
					}
					if cityName, ok := record.City.Names[lang]; ok && cityName != "" {
						idxs = append(idxs, getTagIdx(cityName))
					} else if cityName, ok := record.City.Names["en"]; ok && cityName != "" {
						idxs = append(idxs, getTagIdx(cityName))
					}
				}
				if len(idxs) == 1 {
					idxs = append(idxs, getTagIdx("unknown"))
				}
				addEntryToAggregate(prefix, idxs)
			}
		} else if policy.Meta.Format == "geoip-dat" {
			targetCode := policy.Meta.Config["code"]
			importAll := targetCode == "" || targetCode == "*" || targetCode == "all"

			var data []byte
			data, err = os.ReadFile(tempSrc.Name())
			if err != nil {
				return err
			}

			var v2Entries []parsedV2RayEntry
			v2Entries, err = ParseV2RayGeoIP(data, targetCode, importAll)
			if err != nil {
				return err
			}

			for _, e := range v2Entries {
				addEntryToAggregate(e.Prefix, []uint32{internalTagIdx, getTagIdx(e.CountryCode)})
			}
		} else if policy.Meta.Format == "csv" {
			var f *os.File
			f, err = os.Open(tempSrc.Name())
			if err != nil {
				return err
			}
			defer f.Close()

			sep := policy.Meta.Config["separator"]
			if sep == "" {
				sep = ","
			}

			ipIdx, _ := strconv.Atoi(policy.Meta.Config["ipColumn"])
			tagIdxStr, tagOk := policy.Meta.Config["tagColumn"]
			tagCol := -1
			if tagOk && tagIdxStr != "" {
				tagCol, _ = strconv.Atoi(tagIdxStr)
			}

			// 全局附加标签 (支持多个，逗号分隔)
			additionalTags := strings.Split(policy.Meta.Config["tags"], ",")
			var globalTagIdxs []uint32
			for _, t := range additionalTags {
				t = strings.TrimSpace(t)
				if t != "" {
					globalTagIdxs = append(globalTagIdxs, getTagIdx(t))
				}
			}

			reader := csv.NewReader(f)
			reader.Comma = rune(sep[0])
			reader.FieldsPerRecord = -1

			var records [][]string
			records, err = reader.ReadAll()
			if err != nil {
				return fmt.Errorf("failed to read csv: %w", err)
			}

			for _, row := range records {
				select {
				case <-taskCtx.Done():
					finalErr = context.Canceled
					return finalErr
				default:
				}

				if len(row) <= ipIdx {
					continue
				}
				ipStr := strings.TrimSpace(row[ipIdx])
				if common.IsComment(ipStr) {
					continue
				}

				prefix, err := netip.ParsePrefix(ipStr)
				if err != nil {
					addr, err := netip.ParseAddr(ipStr)
					if err != nil {
						continue
					}
					prefix = netip.PrefixFrom(addr, addr.BitLen())
				}

				idxs := []uint32{internalTagIdx}
				idxs = append(idxs, globalTagIdxs...)

				if tagCol >= 0 && len(row) > tagCol {
					tagVal := strings.TrimSpace(row[tagCol])
					if tagVal != "" {
						idxs = append(idxs, getTagIdx(tagVal))
					}
				}
				addEntryToAggregate(prefix, idxs)
			}
		} else {
			var data []byte
			data, err = os.ReadFile(tempSrc.Name())
			if err != nil {
				return err
			}

			// 全局附加标签 (支持多个，逗号分隔)
			additionalTags := strings.Split(policy.Meta.Config["tags"], ",")
			var globalTagIdxs []uint32
			for _, t := range additionalTags {
				t = strings.TrimSpace(t)
				if t != "" {
					globalTagIdxs = append(globalTagIdxs, getTagIdx(t))
				}
			}
			// 如果没设 tags，默认用 sync
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
				prefix, err := netip.ParsePrefix(line)
				if err != nil {
					addr, err := netip.ParseAddr(line)
					if err != nil {
						continue
					}
					prefix = netip.PrefixFrom(addr, addr.BitLen())
				}
				newIdxs := append([]uint32{internalTagIdx}, globalTagIdxs...)
				addEntryToAggregate(prefix, newIdxs)
			}
		}

		if len(aggregate) == 0 {
			finalErr = fmt.Errorf("no valid IP/CIDR found in source")
			return finalErr
		}

		// 3. 写入目标池
		var release func()
		release, err = s.lockPool(taskCtx, policy.Meta.TargetGroupID)
		if err != nil {
			finalErr = err
			return err
		}
		defer release()

		var group *ipmodel.IPPool
		group, err = repo.GetPool(taskCtx, policy.Meta.TargetGroupID)
		if err != nil {
			finalErr = err
			return err
		}

		_ = s.deps.FS.MkdirAll(PoolsDir, 0755)
		poolPath := filepath.Join(PoolsDir, policy.Meta.TargetGroupID+".bin")
		tempFile := filepath.Join(PoolsDir, policy.Meta.TargetGroupID+".bin.tmp")

		// 处理旧数据合并
		if exists, _ := afero.Exists(s.deps.FS, poolPath); exists {
			var pf afero.File
			pf, err = s.deps.FS.Open(poolPath)
			if err == nil {
				reader, _ := NewReader(pf)
				oldTags := reader.Tags()
				targetInternalTag := strings.ToLower(policy.ID)

				for {
					prefix, tagIdxs, err := reader.NextIndices()
					if err == io.EOF {
						break
					}

					// 检查该条目是否属于当前策略
					isFromThisPolicy := false
					var otherTags []string
					hasOtherPolicy := false

					for _, idx := range tagIdxs {
						tagName := strings.ToLower(oldTags[idx])
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
						// 覆盖模式逻辑：
						if !hasOtherPolicy {
							// 1. 如果该 CIDR 仅由当前策略维护，则直接跳过（不加入聚合器）
							// 这样该 CIDR 的旧标签（包括被删除的标签）都会消失
							continue
						} else {
							// 2. 如果该 CIDR 还有其他策略在引用，则仅移除当前策略的 ID
							// 并将剩余部分加入聚合器，以保护其他策略的数据
							var remainingIdxs []uint32
							for _, ot := range otherTags {
								remainingIdxs = append(remainingIdxs, getTagIdx(ot))
							}
							addEntryToAggregate(prefix, remainingIdxs)
							continue
						}
					}

					// 追加模式或非本策略数据：正常转换并加入聚合
					var convertedIdxs []uint32
					for _, idx := range tagIdxs {
						convertedIdxs = append(convertedIdxs, getTagIdx(oldTags[idx]))
					}
					addEntryToAggregate(prefix, convertedIdxs)
				}
				pf.Close()
			}
		}

		// 转换为最终 Entry 列表
		var finalEntries []Entry
		for prefix, tagSet := range aggregate {
			var idxs []uint32
			for idx := range tagSet {
				idxs = append(idxs, idx)
			}
			finalEntries = append(finalEntries, Entry{Prefix: prefix, TagIndices: idxs})
		}

		finalTags := newTags
		// 写入最终文件
		var tf afero.File
		tf, err = s.deps.FS.Create(tempFile)
		if err != nil {
			finalErr = err
			return err
		}
		codec := NewCodec()
		err = codec.WritePool(tf, finalTags, finalEntries)
		tf.Close()
		if err != nil {
			_ = s.deps.FS.Remove(tempFile)
			finalErr = err
			return err
		}

		// 计算哈希并更新元数据
		var content []byte
		content, err = afero.ReadFile(s.deps.FS, tempFile)
		if err != nil {
			_ = s.deps.FS.Remove(tempFile)
			finalErr = err
			return err
		}
		hf := sha256.New()
		hf.Write(content)

		group.Status.EntryCount = int64(len(finalEntries))
		group.Status.UpdatedAt = time.Now()
		group.Status.Checksum = hex.EncodeToString(hf.Sum(nil))

		if err := s.deps.FS.Rename(tempFile, poolPath); err != nil {
			finalErr = err
			return err
		}

		err = repo.UpdatePoolStatus(taskCtx, group.ID, func(s *ipmodel.IPPoolV1Status) {
			s.EntryCount = int64(len(finalEntries))
			s.UpdatedAt = time.Now()
			s.Checksum = hex.EncodeToString(hf.Sum(nil))
		})

		if err == nil {
			notifyIPPoolChanged(taskCtx, group.ID)
		}
		finalErr = err
		return err
	})
	return finalErr
}
