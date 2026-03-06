package ip

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
	"homelab/pkg/services/discovery"
	"homelab/pkg/services/rbac"
	"io"
	"net/netip"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
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
	}, []string{"get", "list", "create", "update", "delete", "*"})

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
		if limit <= 0 { limit = 20 }
		if offset >= total { return []models.LookupItem{}, total, nil }
		end := offset + limit
		if end > total { end = total }
		return items[offset:end], total, nil
	})
}

const (
	PoolsDir      = "network/ip/pools"
	MaxPoolEntries = 2000000
)

type IPPoolService struct {
	mmdb *MMDBManager
}

func NewIPPoolService(mmdb *MMDBManager) *IPPoolService {
	return &IPPoolService{
		mmdb: mmdb,
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

	// 1. 读取现有数据
	var entries []Entry
	var allTags []string
	tagSet := make(map[string]struct{})

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
				
				// mode 逻辑：过滤掉要删除/修改的目标，保留其它
				if prefix == targetPrefix {
					if mode == "add" {
						pf.Close()
						return fmt.Errorf("ip/cidr already exists: %s", targetPrefix.String())
					}
					if mode == "delete" {
						continue // 跳过即删除
					}
					if mode == "update" {
						continue // 跳过旧的，后面统一添加新的
					}
				}

				// 重建索引：为了简单，直接收集所有标签
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

	if mode == "add" || mode == "update" {
		var tagIndices []uint32
		for _, t := range req.Tags {
			if _, ok := tagSet[t]; !ok {
				tagSet[t] = struct{}{}
				allTags = append(allTags, t)
			}
			idx := slices.Index(allTags, t)
			tagIndices = append(tagIndices, uint32(idx))
		}
		entries = append(entries, Entry{Prefix: targetPrefix, TagIndices: tagIndices})
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
	
	actionName := "AddPoolEntry"
	if mode == "update" { actionName = "UpdatePoolEntry" }
	if mode == "delete" { actionName = "DeletePoolEntry" }
	commonaudit.FromContext(ctx).Log(actionName, targetPrefix.String(), "Success", err == nil)
	
	return err
}

func (s *IPPoolService) DeleteGroup(ctx context.Context, id string) error {
	resource := "network/ip/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
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

	// 删除 DB 记录
	err = repo.DeleteGroup(ctx, id)
	if err != nil {
		return err
	}

	// 级联删除 VFS 中的数据文件
	poolPath := filepath.Join(PoolsDir, id+".bin")
	_ = common.FS.Remove(poolPath)

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
		if perms.IsAllowed("network/ip") || perms.IsAllowed("network/ip/" + g.ID) {
			filtered = append(filtered, g)
		}
	}
	
	total := len(filtered)
	start := (page - 1) * pageSize
	if start < 0 { start = 0 }
	if start >= total { return []models.IPGroup{}, total, nil }
	end := start + pageSize
	if end > total { end = total }
	
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
	if export.ID == "" {
		export.ID = uuid.NewString()
	}
	export.CreatedAt = time.Now()
	export.UpdatedAt = time.Now()
	return repo.SaveExport(ctx, export)
}

func (s *IPPoolService) UpdateExport(ctx context.Context, export *models.IPExport) error {
	old, err := repo.GetExport(ctx, export.ID)
	if err != nil {
		return err
	}
	export.CreatedAt = old.CreatedAt
	export.UpdatedAt = time.Now()
	return repo.SaveExport(ctx, export)
}

func (s *IPPoolService) DeleteExport(ctx context.Context, id string) error {
	return repo.DeleteExport(ctx, id)
}

func (s *IPPoolService) GetExport(ctx context.Context, id string) (*models.IPExport, error) {
	return repo.GetExport(ctx, id)
}

func (s *IPPoolService) ListExports(ctx context.Context, page, pageSize int, search string) ([]models.IPExport, int, error) {
	return repo.ListExports(ctx, page, pageSize, search)
}

// Discovery LookupFuncs

func (s *IPPoolService) LookupGroup(ctx context.Context, id string) (interface{}, error) {
	return repo.GetGroup(ctx, id)
}

func (s *IPPoolService) LookupExport(ctx context.Context, id string) (interface{}, error) {
	return repo.GetExport(ctx, id)
}
