package ip

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
	"homelab/pkg/services/discovery"
	"homelab/pkg/services/rbac"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/oschwald/maxminddb-golang/v2"
	"github.com/robfig/cron/v3"
	"github.com/spf13/afero"
)

var poolWriteMutex sync.Mutex

func init() {
	rbac.RegisterResourceWithVerbs("network/ip", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		res := make([]models.DiscoverResult, 0)
		groups, _, err := repo.ListGroups(ctx, 1, 1000, "")
		if err != nil {
			return nil, err
		}
		for _, g := range groups {
			if strings.HasPrefix(g.ID, prefix) || strings.HasPrefix(g.Name, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: g.ID,
					Name:   g.Name,
					Final:  true,
				})
			}
		}
		return res, nil
	}, []string{"get", "list", "create", "update", "delete", "execute", "*"})

	discovery.Register("network/ip/pools", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		groups, _, err := repo.ListGroups(ctx, 1, 1000, search)
		if err != nil {
			return nil, 0, err
		}
		var items []models.LookupItem
		for _, g := range groups {
			items = append(items, models.LookupItem{
				ID:          g.ID,
				Name:        g.Name,
				Description: g.Description,
			})
		}
		total := len(items)
		if limit <= 0 {
			limit = 20
		}
		if offset >= total {
			return []models.LookupItem{}, total, nil
		}
		end := offset + limit
		if end > total {
			end = total
		}
		return items[offset:end], total, nil
	})
}

const (
	PoolsDir       = "network/ip/pools"
	MaxPoolEntries = 2000000
)

type IPPoolService struct {
	mmdb           *MMDBManager
	cron           *cron.Cron
	cronIDs        map[string]cron.EntryID
	cronLock       sync.Mutex
	exportManager  *ExportManager
	analysisEngine *AnalysisEngine
}

func NewIPPoolService(mmdb *MMDBManager) *IPPoolService {
	return &IPPoolService{
		mmdb:    mmdb,
		cron:    cron.New(),
		cronIDs: make(map[string]cron.EntryID),
	}
}

func (s *IPPoolService) SetExportManager(em *ExportManager) {
	s.exportManager = em
}

func (s *IPPoolService) SetAnalysisEngine(ae *AnalysisEngine) {
	s.analysisEngine = ae
}

func (s *IPPoolService) StartSyncRunner(ctx context.Context) {
	s.cron.Start()
	// 加载所有启用的策略
	// 注入一个系统权限的 context
	sysCtx := commonauth.WithAuth(ctx, &commonauth.AuthContext{
		Type: "sa",
		ID:   "system",
	})
	sysCtx = commonauth.WithPermissions(sysCtx, &models.ResourcePermissions{AllowedAll: true})
	policies, _, _ := repo.ListSyncPolicies(sysCtx, 1, 10000, "")
	for _, p := range policies {
		if p.Enabled {
			s.addCronJob(p)
		}
	}
}

func (s *IPPoolService) addCronJob(p models.IPSyncPolicy) {
	s.cronLock.Lock()
	defer s.cronLock.Unlock()

	// 如果已存在，先删除
	if id, ok := s.cronIDs[p.ID]; ok {
		s.cron.Remove(id)
	}

	id, err := s.cron.AddFunc(p.Cron, func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		// 注入一个系统权限的 context
		ctx = commonauth.WithAuth(ctx, &commonauth.AuthContext{
			Type: "sa",
			ID:   "system",
		})
		ctx = commonauth.WithPermissions(ctx, &models.ResourcePermissions{AllowedAll: true})
		_ = s.Sync(ctx, p.ID)
	})
	if err == nil {
		s.cronIDs[p.ID] = id
	}
}

func (s *IPPoolService) removeCronJob(id string) {
	s.cronLock.Lock()
	defer s.cronLock.Unlock()
	if entryID, ok := s.cronIDs[id]; ok {
		s.cron.Remove(entryID)
		delete(s.cronIDs, id)
	}
}

// Group Methods

func (s *IPPoolService) CreateGroup(ctx context.Context, group *models.IPGroup) error {
	if group.ID == "" {
		group.ID = uuid.NewString()
	}
	resource := "network/ip/" + group.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	err := repo.SaveGroup(ctx, group)
	commonaudit.FromContext(ctx).Log("CreateIPGroup", group.Name, "Created", err == nil)
	return err
}

func (s *IPPoolService) UpdateGroup(ctx context.Context, group *models.IPGroup) error {
	resource := "network/ip/" + group.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	old, err := repo.GetGroup(ctx, group.ID)
	if err != nil {
		return err
	}
	group.CreatedAt = old.CreatedAt
	group.UpdatedAt = time.Now()
	err = repo.SaveGroup(ctx, group)
	commonaudit.FromContext(ctx).Log("UpdateIPGroup", group.Name, "Updated", err == nil)
	return err
}

func (s *IPPoolService) ManagePoolEntry(ctx context.Context, groupID string, req *models.IPPoolEntryRequest, mode string) error {
	resource := "network/ip/" + groupID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	poolWriteMutex.Lock()
	defer poolWriteMutex.Unlock()

	group, err := repo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	targetPrefix, err := netip.ParsePrefix(req.CIDR)
	if err != nil {
		targetAddr, err := netip.ParseAddr(req.CIDR)
		if err != nil {
			return fmt.Errorf("invalid CIDR or IP: %s", req.CIDR)
		}
		targetPrefix = netip.PrefixFrom(targetAddr, targetAddr.BitLen())
	}

	// 保护策略：不允许删除或修改下划线开头的内部标签
	if mode == "update" || mode == "delete" {
		for _, t := range req.OldTags {
			if strings.HasPrefix(t, "_") {
				return fmt.Errorf("internal tag '%s' (starting with '_') cannot be modified or deleted via this interface", t)
			}
		}
	}

	// 1. 读取现有数据并根据 mode 处理
	var entries []Entry
	var allTags []string
	tagSet := make(map[string]struct{})
	found := false

	poolPath := filepath.Join(PoolsDir, groupID+".bin")
	if exists, _ := afero.Exists(common.FS, poolPath); exists {
		pf, err := common.FS.Open(poolPath)
		if err == nil {
			reader, _ := NewReader(pf)
			for {
				prefix, tags, err := reader.Next()
				if err == io.EOF {
					break
				}

				if prefix == targetPrefix {
					found = true
					if mode == "delete" {
						// 仅允许删除没有内部标签的条目，或者仅删除特定非内部标签
						if len(req.OldTags) == 0 {
							hasInternal := false
							for _, t := range tags {
								if strings.HasPrefix(t, "_") {
									hasInternal = true
									break
								}
							}
							if hasInternal {
								pf.Close()
								return fmt.Errorf("entry has internal tags and cannot be deleted entirely via this interface")
							}
							continue // 无内部标签，彻底删除
						} else {
							// 过滤掉指定的旧标签 (不处理内部标签，由 Bind 保证)
							newTagsList := make([]string, 0)
							for _, t := range tags {
								shouldDelete := false
								for _, old := range req.OldTags {
									if t == old {
										shouldDelete = true
										break
									}
								}
								if !shouldDelete || strings.HasPrefix(t, "_") {
									newTagsList = append(newTagsList, t)
								}
							}
							if len(newTagsList) == 0 {
								continue // 标签删完了，删除条目
							}
							tags = newTagsList
						}
					} else if mode == "update" || mode == "add" {
						// 1. 提取当前所有标签
						currentTags := tags
						// 2. 如果是 update 模式，先移除指定的 oldTags
						if mode == "update" {
							filtered := make([]string, 0)
							for _, ct := range currentTags {
								shouldRemove := false
								for _, ot := range req.OldTags {
									if ct == ot {
										shouldRemove = true
										break
									}
								}
								// 即使在 oldTags 中，内部标签也不能被移除 (虽然上层已校验，这里做二次防护)
								if shouldRemove && !strings.HasPrefix(ct, "_") {
									continue
								}
								filtered = append(filtered, ct)
							}
							currentTags = filtered
						}
						// 3. 加入新标签
						currentTags = append(currentTags, req.NewTags...)
						// 4. 去重与排序
						slices.Sort(currentTags)
						tags = slices.Compact(currentTags)
					}
				}

				// 收集并转换索引
				var tagIndices []uint32
				for _, t := range tags {
					if _, ok := tagSet[t]; !ok {
						tagSet[t] = struct{}{}
						allTags = append(allTags, t)
					}
					idx := slices.Index(allTags, t)
					tagIndices = append(tagIndices, uint32(idx))
				}
				entries = append(entries, Entry{Prefix: prefix, TagIndices: tagIndices})
			}
			pf.Close()
		}
	}

	if !found && mode == "add" {
		// 新条目：仅能带入非内部标签 (由 Bind 保证)
		tags := req.NewTags
		var tagIndices []uint32
		for _, t := range tags {
			if _, ok := tagSet[t]; !ok {
				tagSet[t] = struct{}{}
				allTags = append(allTags, t)
			}
			idx := slices.Index(allTags, t)
			tagIndices = append(tagIndices, uint32(idx))
		}
		entries = append(entries, Entry{Prefix: targetPrefix, TagIndices: tagIndices})
	} else if !found && (mode == "update" || mode == "delete") {
		return fmt.Errorf("ip/cidr not found: %s", targetPrefix.String())
	}

	// 2. 写回
	_ = common.FS.MkdirAll(PoolsDir, 0755)
	tempFile := filepath.Join(PoolsDir, groupID+".bin.tmp")
	tf, err := common.FS.Create(tempFile)
	if err != nil {
		return err
	}
	codec := NewCodec()
	err = codec.WritePool(tf, allTags, entries)
	tf.Close()
	if err != nil {
		return err
	}
	_ = common.FS.Rename(tempFile, poolPath)

	// 3. 更新元数据
	group.EntryCount = int64(len(entries))
	group.UpdatedAt = time.Now()

	hf := sha256.New()
	content, _ := afero.ReadFile(common.FS, poolPath)
	hf.Write(content)
	group.Checksum = hex.EncodeToString(hf.Sum(nil))

	err = repo.SaveGroup(ctx, group)
	if err == nil {
		if s.analysisEngine != nil {
			s.analysisEngine.RemoveCache(groupID)
		}
	}

	actionName := "ManagePoolEntry"
	commonaudit.FromContext(ctx).Log(actionName, targetPrefix.String(), mode, err == nil)

	return err
}

func (s *IPPoolService) DeleteGroup(ctx context.Context, id string) error {
	resource := "network/ip/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	poolWriteMutex.Lock()
	defer poolWriteMutex.Unlock()

	old, _ := repo.GetGroup(ctx, id)
	// 校验依赖：检查是否有 IPExport 引用了此池
	exports, _, err := repo.ListExports(ctx, 1, 1000, "")
	if err != nil {
		return err
	}
	for _, e := range exports {
		if slices.Contains(e.GroupIDs, id) {
			return fmt.Errorf("cannot delete group %s: referenced by export %s", id, e.Name)
		}
	}

	// 校验依赖：检查是否有同步策略引用了此池
	policies, _, err := repo.ListSyncPolicies(ctx, 1, 1000, "")
	if err != nil {
		return err
	}
	for _, p := range policies {
		if p.TargetGroupID == id {
			return fmt.Errorf("cannot delete group %s: referenced by sync policy %s", id, p.Name)
		}
	}

	// 删除 DB 记录
	err = repo.DeleteGroup(ctx, id)
	if err != nil {
		return err
	}

	// 级联删除 VFS 中的数据文件
	poolPath := filepath.Join(PoolsDir, id+".bin")
	_ = common.FS.Remove(poolPath)
	if s.analysisEngine != nil {
		s.analysisEngine.RemoveCache(id)
	}

	if old != nil {
		commonaudit.FromContext(ctx).Log("DeleteIPGroup", old.Name, "Deleted", true)
	}
	return nil
}

func (s *IPPoolService) GetGroup(ctx context.Context, id string) (*models.IPGroup, error) {
	resource := "network/ip/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) && !commonauth.PermissionsFromContext(ctx).IsAllowed("network/ip") {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	return repo.GetGroup(ctx, id)
}

func (s *IPPoolService) ListGroups(ctx context.Context, page, pageSize int, search string) ([]models.IPGroup, int, error) {
	groups, _, err := repo.ListGroups(ctx, 1, 10000, search)
	if err != nil {
		return nil, 0, err
	}

	var filtered []models.IPGroup
	perms := commonauth.PermissionsFromContext(ctx)
	for _, g := range groups {
		if perms.IsAllowed("network/ip") || perms.IsAllowed("network/ip/"+g.ID) {
			filtered = append(filtered, g)
		}
	}

	total := len(filtered)
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []models.IPGroup{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	return filtered[start:end], total, nil
}

// Preview Method

func (s *IPPoolService) PreviewPool(ctx context.Context, groupID string, cursor int64, limit int, search string) (*models.IPPoolPreviewResponse, error) {
	resource := "network/ip/" + groupID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) && !commonauth.PermissionsFromContext(ctx).IsAllowed("network/ip") {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	poolPath := filepath.Join(PoolsDir, groupID+".bin")
	f, err := common.FS.Open(poolPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	reader, err := NewReader(f)
	if err != nil {
		return nil, err
	}

	if cursor > 0 {
		if seeker, ok := f.(io.Seeker); ok {
			if _, err := seeker.Seek(cursor, io.SeekStart); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("vfs does not support seeking")
		}
	}

	res := &models.IPPoolPreviewResponse{
		Total:   int64(reader.EntryCount()),
		Entries: make([]models.IPPoolEntry, 0),
	}

	search = strings.ToLower(search)
	matched := 0

	for {
		if matched >= limit {
			break
		}

		prefix, tags, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if search != "" {
			prefixStr := prefix.String()
			matchFound := strings.Contains(strings.ToLower(prefixStr), search)
			if !matchFound {
				for _, t := range tags {
					if strings.Contains(strings.ToLower(t), search) {
						matchFound = true
						break
					}
				}
			}
			if !matchFound {
				continue
			}
		}

		res.Entries = append(res.Entries, models.IPPoolEntry{
			CIDR: prefix.String(),
			Tags: tags,
		})
		matched++
	}

	// 获取当前文件偏移量作为下一个 cursor
	if seeker, ok := f.(io.Seeker); ok {
		next, _ := seeker.Seek(0, io.SeekCurrent)
		if matched < limit {
			res.NextCursor = 0
		} else {
			res.NextCursor = next
		}
	}

	return res, nil
}

// Export Methods

func (s *IPPoolService) CreateExport(ctx context.Context, export *models.IPExport) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/ip") {
		return fmt.Errorf("%w: network/ip", commonauth.ErrPermissionDenied)
	}
	if export.ID == "" {
		export.ID = uuid.NewString()
	}
	export.CreatedAt = time.Now()
	export.UpdatedAt = time.Now()
	err := repo.SaveExport(ctx, export)
	commonaudit.FromContext(ctx).Log("CreateIPExport", export.Name, "Created", err == nil)
	return err
}

func (s *IPPoolService) UpdateExport(ctx context.Context, export *models.IPExport) error {
	resource := "network/ip/export/" + export.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	old, err := repo.GetExport(ctx, export.ID)
	if err != nil {
		return err
	}
	export.CreatedAt = old.CreatedAt
	export.UpdatedAt = time.Now()
	err = repo.SaveExport(ctx, export)
	commonaudit.FromContext(ctx).Log("UpdateIPExport", export.Name, "Updated", err == nil)
	return err
}

func (s *IPPoolService) DeleteExport(ctx context.Context, id string) error {
	resource := "network/ip/export/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	// 级联删除相关的任务和物理文件
	if s.exportManager != nil {
		s.exportManager.DeleteTasksByExportID(id)
	}
	err := repo.DeleteExport(ctx, id)
	commonaudit.FromContext(ctx).Log("DeleteIPExport", id, "Deleted", err == nil)
	return err
}

func (s *IPPoolService) GetExport(ctx context.Context, id string) (*models.IPExport, error) {
	return repo.GetExport(ctx, id)
}

func (s *IPPoolService) ListExports(ctx context.Context, page, pageSize int, search string) ([]models.IPExport, int, error) {
	return repo.ListExports(ctx, page, pageSize, search)
}

func (s *IPPoolService) PreviewExport(ctx context.Context, req *models.IPExportPreviewRequest) ([]models.IPPoolEntry, error) {
	program, err := expr.Compile(req.Rule, expr.Env(map[string]interface{}{
		"tags": []string{},
		"cidr": "",
		"ip":   "",
	}))
	if err != nil {
		return nil, fmt.Errorf("%w: compile error: %v", common.ErrBadRequest, err)
	}

	var results []models.IPPoolEntry
	for _, gid := range req.GroupIDs {
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

		for len(results) < 50 {
			prefix, tags, err := reader.Next()
			if err == io.EOF {
				break
			}

			output, err := expr.Run(program, map[string]interface{}{
				"tags": tags,
				"cidr": prefix.String(),
				"ip":   prefix.Addr().String(),
			})

			if err == nil && output == true {
				results = append(results, models.IPPoolEntry{
					CIDR: prefix.String(),
					Tags: tags,
				})
			}
		}
		pf.Close()
		if len(results) >= 50 {
			break
		}
	}

	return results, nil
}

// Discovery LookupFuncs

func (s *IPPoolService) LookupGroup(ctx context.Context, id string) (interface{}, error) {
	return repo.GetGroup(ctx, id)
}

func (s *IPPoolService) LookupExport(ctx context.Context, id string) (interface{}, error) {
	return repo.GetExport(ctx, id)
}

// Sync Methods

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
	}
	commonaudit.FromContext(ctx).Log("DeleteIPSyncPolicy", old.Name, "Deleted", err == nil)
	return err
}

func (s *IPPoolService) GetSyncPolicy(ctx context.Context, id string) (*models.IPSyncPolicy, error) {
	return repo.GetSyncPolicy(ctx, id)
}

func (s *IPPoolService) ListSyncPolicies(ctx context.Context, page, pageSize int, search string) ([]models.IPSyncPolicy, int, error) {
	return repo.ListSyncPolicies(ctx, page, pageSize, search)
}

func (s *IPPoolService) Sync(ctx context.Context, id string) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("network/ip") {
		return fmt.Errorf("%w: network/ip", commonauth.ErrPermissionDenied)
	}
	policy, err := repo.GetSyncPolicy(ctx, id)
	if err != nil {
		return err
	}

	policy.LastRunAt = time.Now()

	err = s.doSync(ctx, policy)
	if err != nil {
		policy.LastStatus = "failed"
		policy.ErrorMessage = err.Error()
	} else {
		policy.LastStatus = "success"
		policy.ErrorMessage = ""
	}

	_ = repo.SaveSyncPolicy(ctx, policy)
	commonaudit.FromContext(ctx).Log("TriggerIPSync", policy.Name, "Triggered", err == nil)
	return err
}

func (s *IPPoolService) doSync(ctx context.Context, policy *models.IPSyncPolicy) error {
	if policy == nil {
		return fmt.Errorf("policy is nil")
	}
	if policy.ID == "" {
		policy.ID = generatePolicyID()
	}
	if policy.Config == nil {
		policy.Config = make(map[string]string)
	}

	// 1. 下载原始数据到临时文件
	// SSRF 防护：校验 URL
	if err := validateSourceURL(policy.SourceURL, policy); err != nil {
		return err
	}

	tempSrc, err := os.CreateTemp("", "sync_src_*.tmp")
	if err != nil {
		return err
	}
	defer os.Remove(tempSrc.Name())
	defer tempSrc.Close()

	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(policy.SourceURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if _, err := io.Copy(tempSrc, resp.Body); err != nil {
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
		mdb, err := maxminddb.Open(tempSrc.Name())
		if err != nil {
			return fmt.Errorf("failed to open as maxminddb: %w", err)
		}
		defer mdb.Close()

		for result := range mdb.Networks() {
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

		data, err := os.ReadFile(tempSrc.Name())
		if err != nil {
			return err
		}

		v2Entries, err := ParseV2RayGeoIP(data, targetCode, importAll)
		if err != nil {
			return err
		}

		for _, e := range v2Entries {
			addEntryToAggregate(e.Prefix, []uint32{internalTagIdx, getTagIdx(e.CountryCode)})
		}
	} else if policy.Format == "csv" {
		f, err := os.Open(tempSrc.Name())
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

		records, err := reader.ReadAll()
		if err != nil {
			return fmt.Errorf("failed to read csv: %w", err)
		}

		for _, row := range records {
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
		data, err := os.ReadFile(tempSrc.Name())
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
		return fmt.Errorf("no valid IP/CIDR found in source")
	}

	// 3. 写入目标池
	poolWriteMutex.Lock()
	defer poolWriteMutex.Unlock()

	group, err := repo.GetGroup(ctx, policy.TargetGroupID)
	if err != nil {
		return err
	}

	_ = common.FS.MkdirAll(PoolsDir, 0755)
	poolPath := filepath.Join(PoolsDir, policy.TargetGroupID+".bin")
	tempFile := filepath.Join(PoolsDir, policy.TargetGroupID+".bin.tmp")

	// 处理旧数据合并
	if exists, _ := afero.Exists(common.FS, poolPath); exists {
		pf, err := common.FS.Open(poolPath)
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
	tf, err := common.FS.Create(tempFile)
	if err != nil {
		return err
	}
	codec := NewCodec()
	err = codec.WritePool(tf, finalTags, finalEntries)
	tf.Close()
	if err != nil {
		return err
	}
	_ = common.FS.Rename(tempFile, poolPath)

	// 更新元数据
	group.EntryCount = int64(len(finalEntries))
	group.UpdatedAt = time.Now()
	hf := sha256.New()
	content, _ := afero.ReadFile(common.FS, poolPath)
	hf.Write(content)
	group.Checksum = hex.EncodeToString(hf.Sum(nil))

	return repo.SaveGroup(ctx, group)
}

func generatePolicyID() string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, 10)
	b[0] = '_'
	rb := make([]byte, 9)
	_, _ = rand.Read(rb)
	for i := 0; i < 9; i++ {
		b[i+1] = letters[rb[i]%uint8(len(letters))]
	}
	return string(b)
}
func validateSourceURL(urlStr string, policy *models.IPSyncPolicy) error {
	allowPrivate := false
	if policy != nil && policy.Config != nil && policy.Config["allowPrivate"] == "true" {
		allowPrivate = true
	}

	u, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	hostname := u.Hostname()
	if hostname == "localhost" && !allowPrivate {
		return fmt.Errorf("SSRF detected: localhost is forbidden")
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		// 如果解析失败，可能是 IP 直连或者非法的
		ip := net.ParseIP(hostname)
		if ip != nil {
			if isPrivateIP(ip) && !allowPrivate {
				return fmt.Errorf("SSRF detected: private IP %s is forbidden", ip)
			}
			return nil
		}
		// 暂时允许无法解析的情况（如容器内 DNS），但在生产中应更严格
		return nil
	}

	for _, ip := range ips {
		if isPrivateIP(ip) && !allowPrivate {
			return fmt.Errorf("SSRF detected: host %s resolves to private IP %s", hostname, ip)
		}
	}

	return nil
}

func isPrivateIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		return ip4[0] == 10 ||
			(ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31) ||
			(ip4[0] == 192 && ip4[1] == 168)
	}
	return false
}
