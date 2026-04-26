package site

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	sitemodel "homelab/pkg/models/network/site"
	repo "homelab/pkg/repositories/network/site"
	"io"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
)

func (s *SitePoolService) ManagePoolEntry(ctx context.Context, groupID string, req *sitemodel.SitePoolEntryRequest, mode string) error {
	if err := requireSiteResource(ctx, siteGroupResource(groupID)); err != nil {
		return err
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

	var entryToUpdate *sitemodel.SitePoolEntry

	poolPath := filepath.Join(PoolsDir, groupID+".bin")
	if exists, _ := afero.Exists(s.deps.FS, poolPath); exists {
		pf, err := s.deps.FS.Open(poolPath)
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
						if mode == "update" {
							entryToUpdate = &entry
						}
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
		finalTags := req.NewTags
		if mode == "update" && entryToUpdate != nil {
			// 只有在提供 OldTags 时才执行增量更新，否则全量替换（保留内部标签）
			merged := make(map[string]struct{})
			// 1. 保留原本的所有内部标签
			for _, t := range entryToUpdate.Tags {
				if strings.HasPrefix(t, "_") {
					merged[t] = struct{}{}
				}
			}
			if len(req.OldTags) > 0 {
				// 增量模式：保留原本非 OldTags 的用户标签
				oldTagSet := make(map[string]struct{})
				for _, t := range req.OldTags {
					oldTagSet[t] = struct{}{}
				}
				for _, t := range entryToUpdate.Tags {
					if !strings.HasPrefix(t, "_") {
						if _, ok := oldTagSet[t]; !ok {
							merged[t] = struct{}{}
						}
					}
				}
			}
			// 2. 加入本次的新标签
			for _, t := range req.NewTags {
				merged[t] = struct{}{}
			}
			finalTags = make([]string, 0, len(merged))
			for t := range merged {
				finalTags = append(finalTags, t)
			}
		}

		var tagIndices []uint32
		for _, t := range finalTags {
			if _, ok := tagSet[t]; !ok {
				tagSet[t] = struct{}{}
				allTags = append(allTags, t)
			}
			tagIndices = append(tagIndices, uint32(slices.Index(allTags, t)))
		}
		entries = append(entries, Entry{Type: req.Type, Value: req.Value, TagIndices: tagIndices})
	}

	_ = s.deps.FS.MkdirAll(PoolsDir, 0755)
	tempFile := poolPath + ".tmp"
	tf, err := s.deps.FS.Create(tempFile)
	if err != nil {
		return err
	}
	codec := NewCodec()
	err = codec.WritePool(tf, allTags, entries)
	tf.Close()
	if err != nil {
		_ = s.deps.FS.Remove(tempFile)
		return err
	}

	// 计算哈希并更新元数据
	content, err := afero.ReadFile(s.deps.FS, tempFile)
	if err != nil {
		_ = s.deps.FS.Remove(tempFile)
		return err
	}
	hf := sha256.New()
	hf.Write(content)

	group.Status.EntryCount = int64(len(entries))
	group.Status.UpdatedAt = time.Now()
	group.Status.Checksum = hex.EncodeToString(hf.Sum(nil))

	if err := s.deps.FS.Rename(tempFile, poolPath); err != nil {
		return err
	}

	_ = repo.SaveGroup(ctx, group)
	if s.engine != nil {
		notifySitePoolChanged(ctx, groupID)
	}

	commonaudit.FromContext(ctx).Log("ManageSiteEntry", req.Value, mode, true)
	return nil
}

func notifySitePoolChanged(ctx context.Context, groupID string) {
	common.UpdateGlobalVersion(ctx, "network/site/pool/"+groupID)
	common.NotifyCluster(ctx, common.EventSitePoolChanged, groupID)
}

func (s *SitePoolService) PreviewPool(ctx context.Context, groupID string, cursorStr string, limit int, search string) (*sitemodel.SitePoolPreviewResponse, error) {
	cursor, _ := strconv.ParseInt(cursorStr, 10, 64)
	if err := requireSiteResourceOrGlobal(ctx, siteGroupResource(groupID)); err != nil {
		return nil, err
	}
	poolPath := filepath.Join(PoolsDir, groupID+".bin")
	f, err := s.deps.FS.Open(poolPath)
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

	res := &sitemodel.SitePoolPreviewResponse{Total: int64(reader.EntryCount()), Entries: []sitemodel.SitePoolEntry{}}
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
		common.SortTags(entry.Tags)
		res.Entries = append(res.Entries, entry)
		matched++
	}

	if seeker, ok := f.(io.Seeker); ok {
		next, _ := seeker.Seek(0, io.SeekCurrent)
		if matched < limit {
			res.NextCursor = ""
		} else {
			res.NextCursor = strconv.FormatInt(next, 10)
		}
	}
	return res, nil
}
