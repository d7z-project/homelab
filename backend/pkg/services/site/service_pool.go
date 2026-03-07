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
	"io"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/afero"
)

func (s *SitePoolService) ManagePoolEntry(ctx context.Context, groupID string, req *models.SitePoolEntryRequest, mode string) error {
	resource := "network/site/" + groupID
	if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
		return fmt.Errorf("%w: %s", commonauth.ErrPermissionDenied, resource)
	}

	release, err := s.lockPool(ctx, groupID)
	if err != nil {
		return err
	}
	defer release()

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
		_ = common.FS.Remove(tempFile)
		return err
	}

	// 计算哈希并更新元数据
	content, err := afero.ReadFile(common.FS, tempFile)
	if err != nil {
		_ = common.FS.Remove(tempFile)
		return err
	}
	hf := sha256.New()
	hf.Write(content)

	group.EntryCount = int64(len(entries))
	group.UpdatedAt = time.Now()
	group.Checksum = hex.EncodeToString(hf.Sum(nil))

	if err := common.FS.Rename(tempFile, poolPath); err != nil {
		return err
	}

	_ = repo.SaveGroup(ctx, group)
	if s.engine != nil {
		notifySitePoolUpdate(ctx, groupID)
	}

	commonaudit.FromContext(ctx).Log("ManageSiteEntry", req.Value, mode, true)
	return nil
}

func notifySitePoolUpdate(ctx context.Context, groupID string) {
	common.UpdateGlobalVersion(ctx, "network/site/pool/"+groupID)
	common.NotifyCluster(ctx, "site_pool_update", groupID)
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
