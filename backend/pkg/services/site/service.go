package site

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/site"
	"homelab/pkg/services/discovery"
	"homelab/pkg/services/rbac"
	"io"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/spf13/afero"
)

var sitePoolWriteMutex sync.Mutex

func init() {
	rbac.RegisterResourceWithVerbs("network/site", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
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

	discovery.Register("network/site/pools", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
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
	PoolsDir = "network/site/pools"
)

type SitePoolService struct {
	engine        *AnalysisEngine
	exportManager *ExportManager
}

func NewSitePoolService(engine *AnalysisEngine) *SitePoolService {
	return &SitePoolService{engine: engine}
}

func (s *SitePoolService) SetExportManager(em *ExportManager) {
	s.exportManager = em
}

// Group Methods

func (s *SitePoolService) CreateGroup(ctx context.Context, group *models.SiteGroup) error {
	group.ID = uuid.NewString()
	resource := "network/site/" + group.ID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	group.CreatedAt = time.Now()
	group.UpdatedAt = time.Now()
	err := repo.SaveGroup(ctx, group)
	commonaudit.FromContext(ctx).Log("CreateSiteGroup", group.Name, "Created", err == nil)
	return err
}

func (s *SitePoolService) UpdateGroup(ctx context.Context, group *models.SiteGroup) error {
	resource := "network/site/" + group.ID
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
	commonaudit.FromContext(ctx).Log("UpdateSiteGroup", group.Name, "Updated", err == nil)
	return err
}

func (s *SitePoolService) DeleteGroup(ctx context.Context, id string) error {
	resource := "network/site/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	sitePoolWriteMutex.Lock()
	defer sitePoolWriteMutex.Unlock()

	old, _ := repo.GetGroup(ctx, id)
	exports, _, err := repo.ListExports(ctx, 1, 1000, "")
	if err != nil {
		return err
	}
	for _, e := range exports {
		if slices.Contains(e.GroupIDs, id) {
			return fmt.Errorf("cannot delete group %s: referenced by export %s", id, e.Name)
		}
	}

	if err := repo.DeleteGroup(ctx, id); err != nil {
		return err
	}

	poolPath := filepath.Join(PoolsDir, id+".bin")
	_ = common.FS.Remove(poolPath)
	if s.engine != nil {
		s.engine.RemoveCache(id)
	}

	if old != nil {
		commonaudit.FromContext(ctx).Log("DeleteSiteGroup", old.Name, "Deleted", true)
	}
	return nil
}

func (s *SitePoolService) GetGroup(ctx context.Context, id string) (*models.SiteGroup, error) {
	resource := "network/site/" + id
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) && !commonauth.PermissionsFromContext(ctx).IsAllowed("network/site") {
		return nil, fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}
	return repo.GetGroup(ctx, id)
}

func (s *SitePoolService) ListGroups(ctx context.Context, page, pageSize int, search string) ([]models.SiteGroup, int, error) {
	groups, _, err := repo.ListGroups(ctx, 1, 10000, search)
	if err != nil {
		return nil, 0, err
	}
	var filtered []models.SiteGroup
	perms := commonauth.PermissionsFromContext(ctx)
	for _, g := range groups {
		if perms.IsAllowed("network/site") || perms.IsAllowed("network/site/"+g.ID) {
			filtered = append(filtered, g)
		}
	}
	total := len(filtered)
	start := (page - 1) * pageSize
	if start < 0 {
		start = 0
	}
	if start >= total {
		return []models.SiteGroup{}, total, nil
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return filtered[start:end], total, nil
}

// Entry Methods

func (s *SitePoolService) ManagePoolEntry(ctx context.Context, groupID string, req *models.SitePoolEntryRequest, mode string) error {
	resource := "network/site/" + groupID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	sitePoolWriteMutex.Lock()
	defer sitePoolWriteMutex.Unlock()

	group, err := repo.GetGroup(ctx, groupID)
	if err != nil {
		return err
	}

	var entries []Entry
	var allTags []string
	tagSet := make(map[string]struct{})

	poolPath := filepath.Join(PoolsDir, groupID+".bin")
	if exists, _ := afero.Exists(common.FS, poolPath); exists {
		pf, err := common.FS.Open(poolPath)
		if err == nil {
			reader, _ := NewReader(pf)
			for {
				entry, err := reader.Next()
				if err == io.EOF {
					break
				}

				if entry.Type == req.Type && entry.Value == req.Value {
					if mode == "add" {
						pf.Close()
						return fmt.Errorf("rule already exists: %d:%s", req.Type, req.Value)
					}
					if mode == "delete" || mode == "update" {
						continue
					}
				}

				var tagIndices []uint32
				for _, t := range entry.Tags {
					if _, ok := tagSet[t]; !ok {
						tagSet[t] = struct{}{}
						allTags = append(allTags, t)
					}
					tagIndices = append(tagIndices, uint32(slices.Index(allTags, t)))
				}
				entries = append(entries, Entry{Type: entry.Type, Value: entry.Value, TagIndices: tagIndices})
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
			tagIndices = append(tagIndices, uint32(slices.Index(allTags, t)))
		}
		entries = append(entries, Entry{Type: req.Type, Value: req.Value, TagIndices: tagIndices})
	}

	_ = common.FS.MkdirAll(PoolsDir, 0755)
	tempFile := poolPath + ".tmp"
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

	group.EntryCount = int64(len(entries))
	group.UpdatedAt = time.Now()
	hf := sha256.New()
	content, _ := afero.ReadFile(common.FS, poolPath)
	hf.Write(content)
	group.Checksum = hex.EncodeToString(hf.Sum(nil))
	_ = repo.SaveGroup(ctx, group)

	commonaudit.FromContext(ctx).Log("ManageSiteEntry", req.Value, mode, true)
	return nil
}

func (s *SitePoolService) PreviewPool(ctx context.Context, groupID string, cursor int64, limit int, search string) (*models.SitePoolPreviewResponse, error) {
	resource := "network/site/" + groupID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) && !commonauth.PermissionsFromContext(ctx).IsAllowed("network/site") {
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

	res := &models.SitePoolPreviewResponse{Total: int64(reader.EntryCount()), Entries: []models.SitePoolEntry{}}
	search = strings.ToLower(search)
	matched := 0
	for {
		if matched >= limit {
			break
		}
		entry, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if search != "" {
			found := strings.Contains(strings.ToLower(entry.Value), search)
			if !found {
				for _, t := range entry.Tags {
					if strings.Contains(strings.ToLower(t), search) {
						found = true
						break
					}
				}
			}
			if !found {
				continue
			}
		}
		res.Entries = append(res.Entries, entry)
		matched++
	}

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

func (s *SitePoolService) CreateExport(ctx context.Context, export *models.SiteExport) error {
	export.ID = uuid.NewString()
	export.CreatedAt = time.Now()
	export.UpdatedAt = time.Now()
	return repo.SaveExport(ctx, export)
}

func (s *SitePoolService) UpdateExport(ctx context.Context, export *models.SiteExport) error {
	old, err := repo.GetExport(ctx, export.ID)
	if err != nil {
		return err
	}
	export.CreatedAt = old.CreatedAt
	export.UpdatedAt = time.Now()
	return repo.SaveExport(ctx, export)
}

func (s *SitePoolService) DeleteExport(ctx context.Context, id string) error {
	if s.exportManager != nil {
		s.exportManager.DeleteTasksByExportID(id)
	}
	return repo.DeleteExport(ctx, id)
}

func (s *SitePoolService) GetExport(ctx context.Context, id string) (*models.SiteExport, error) {
	return repo.GetExport(ctx, id)
}

func (s *SitePoolService) ListExports(ctx context.Context, page, pageSize int, search string) ([]models.SiteExport, int, error) {
	return repo.ListExports(ctx, page, pageSize, search)
}

func (s *SitePoolService) PreviewExport(ctx context.Context, req *models.SiteExportPreviewRequest) ([]models.SitePoolEntry, error) {
	program, err := expr.Compile(req.Rule, expr.Env(map[string]interface{}{
		"tags":   []string{},
		"domain": "",
		"type":   uint8(0),
	}))
	if err != nil {
		return nil, fmt.Errorf("%w: compile error: %v", common.ErrBadRequest, err)
	}

	var results []models.SitePoolEntry
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
			entry, err := reader.Next()
			if err == io.EOF {
				break
			}

			out, err := expr.Run(program, map[string]interface{}{
				"tags":   entry.Tags,
				"domain": entry.Value,
				"type":   entry.Type,
			})

			if err == nil && out == true {
				results = append(results, entry)
			}
		}
		pf.Close()
		if len(results) >= 50 {
			break
		}
	}

	return results, nil
}
