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
	commonauth "homelab/pkg/common/auth"
	taskpkg "homelab/pkg/common/task"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
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

func (s *IPPoolService) CreateSyncPolicy(ctx context.Context, policy *models.IPSyncPolicy) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/ip") {
		return fmt.Errorf("%w: network/ip", commonauth.ErrPermissionDenied)
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
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	err := repo.SaveSyncPolicy(ctx, policy)
	if err == nil && policy.Enabled {
		s.addCronJob(*policy)
		common.NotifyCluster(ctx, "ip_sync_policy_update", policy.ID)
	}
	commonaudit.FromContext(ctx).Log("CreateIPSyncPolicy", policy.Name, "Created", err == nil)
	return err
}

func (s *IPPoolService) UpdateSyncPolicy(ctx context.Context, policy *models.IPSyncPolicy) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/ip") {
		return fmt.Errorf("%w: network/ip", commonauth.ErrPermissionDenied)
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
		common.NotifyCluster(ctx, "ip_sync_policy_update", policy.ID)
	}
	commonaudit.FromContext(ctx).Log("UpdateIPSyncPolicy", policy.Name, "Updated", err == nil)
	return err
}

func (s *IPPoolService) DeleteSyncPolicy(ctx context.Context, id string) error {
	old, err := repo.GetSyncPolicy(ctx, id)
	if err != nil {
		return err
	}
	err = repo.DeleteSyncPolicy(ctx, id)
	if err == nil {
		s.removeCronJob(id)
		common.NotifyCluster(ctx, "ip_sync_policy_delete", id)
	}
	commonaudit.FromContext(ctx).Log("DeleteIPSyncPolicy", old.Name, "Deleted", err == nil)
	return err
}

func (s *IPPoolService) GetSyncPolicy(ctx context.Context, id string) (*models.IPSyncPolicy, error) {
	return repo.GetSyncPolicy(ctx, id)
}

func (s *IPPoolService) ListSyncPolicies(ctx context.Context, page, pageSize int, search string) ([]models.IPSyncPolicy, int, error) {
	list, total, err := repo.ListSyncPolicies(ctx, page, pageSize, search)
	if err != nil {
		return nil, 0, err
	}

	// 从内存 `manager` 获取最新运行状态、错误和进度
	for i := range list {
		if t, ok := s.syncTasks.GetTask(list[i].ID); ok {
			status := t.GetStatus()
			if status == models.TaskStatusRunning || status == models.TaskStatusPending {
				list[i].LastStatus = status
				list[i].ErrorMessage = t.Error
				list[i].Progress = t.GetProgress()
			}
		}
	}
	return list, total, nil
}

func (s *IPPoolService) Sync(ctx context.Context, id string) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/ip") {
		return fmt.Errorf("%w: network/ip", commonauth.ErrPermissionDenied)
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
		if status == models.TaskStatusPending || status == models.TaskStatusRunning {
			lockKey := "action:ip_sync:" + id
			if release := common.Locker.TryLock(ctx, lockKey); release != nil {
				// 任务僵死：框架强制标记取消
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
		common.NotifyCluster(ctx, "ip_sync_run", id)
	} else {
		// Standalone/Test mode fallback: execute locally in background
		go func() {
			// 注入系统权限
			sysCtx := commonauth.WithAuth(context.Background(), &commonauth.AuthContext{
				Type: "sa",
				ID:   "system",
			})
			sysCtx = commonauth.WithPermissions(sysCtx, &models.ResourcePermissions{AllowedAll: true})
			_ = s.doSync(sysCtx, id)
		}()
	}

	commonaudit.FromContext(ctx).Log("TriggerIPSync", policy.Name, "Triggered Asynchronously", true)
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

		policy.LastStatus = models.TaskStatusRunning
		_ = repo.SaveSyncPolicy(taskCtx, policy)

		defer func() {
			if policy == nil {
				return
			}
			// 使用闭包捕获的命名返回值以确保最后的保存
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
			commonaudit.FromContext(taskCtx).Log("IPSyncExecute", policy.Name, "Finished", finalErr == nil)
		}()

		if policy.Config == nil {
			policy.Config = make(map[string]string)
		}

		// 1. 下载原始数据到临时文件
		// SSRF 防护：校验 URL
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

		if policy.Format == "geoip" {
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
					lang := policy.Config["language"]
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
		} else if policy.Format == "geoip-dat" {
			targetCode := policy.Config["code"]
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
		} else if policy.Format == "csv" {
			var f *os.File
			f, err = os.Open(tempSrc.Name())
			if err != nil {
				return err
			}
			defer f.Close()

			sep := policy.Config["separator"]
			if sep == "" {
				sep = ","
			}

			ipIdx, _ := strconv.Atoi(policy.Config["ipColumn"])
			tagIdxStr, tagOk := policy.Config["tagColumn"]
			tagCol := -1
			if tagOk && tagIdxStr != "" {
				tagCol, _ = strconv.Atoi(tagIdxStr)
			}

			// 全局附加标签 (支持多个，逗号分隔)
			additionalTags := strings.Split(policy.Config["tags"], ",")
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
				if ipStr == "" || strings.HasPrefix(ipStr, "#") {
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
			additionalTags := strings.Split(policy.Config["tags"], ",")
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

				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
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
		release, err = s.lockPool(taskCtx, policy.TargetGroupID)
		if err != nil {
			finalErr = err
			return err
		}
		defer release()

		var group *models.IPGroup
		group, err = repo.GetGroup(taskCtx, policy.TargetGroupID)
		if err != nil {
			finalErr = err
			return err
		}

		_ = common.FS.MkdirAll(PoolsDir, 0755)
		poolPath := filepath.Join(PoolsDir, policy.TargetGroupID+".bin")
		tempFile := filepath.Join(PoolsDir, policy.TargetGroupID+".bin.tmp")

		// 处理旧数据合并
		if exists, _ := afero.Exists(common.FS, poolPath); exists {
			var pf afero.File
			pf, err = common.FS.Open(poolPath)
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

					if isFromThisPolicy && policy.Mode == "overwrite" {
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

		// 计算哈希并更新元数据
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
			notifyIPPoolUpdate(taskCtx, group.ID)
		}
		finalErr = err
		return err
	})
	return finalErr
}
