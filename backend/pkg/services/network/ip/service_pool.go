package ip

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	ipmodel "homelab/pkg/models/network/ip"
	repo "homelab/pkg/repositories/network/ip"
	"io"
	"net/netip"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
)

func (s *IPPoolService) ManagePoolEntry(ctx context.Context, groupID string, req *ipmodel.IPPoolEntryRequest, mode string) error {
	if err := requireIPResource(ctx, ipGroupResource(groupID)); err != nil {
		return err
	}

	release, err := s.lockPool(ctx, groupID)
	if err != nil {
		return err
	}
	defer release()

	group, err := repo.PoolRepo.Get(ctx, groupID)
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
		_ = common.FS.Remove(tempFile)
		return err
	}

	// 计算哈希并更新元数据（在 Rename 之前，确保元数据与即将生效的文件一致）
	content, err := afero.ReadFile(common.FS, tempFile)
	if err != nil {
		_ = common.FS.Remove(tempFile)
		return err
	}
	hf := sha256.New()
	hf.Write(content)

	if err := common.FS.Rename(tempFile, poolPath); err != nil {
		return err
	}

	err = repo.PoolRepo.UpdateStatus(ctx, group.ID, func(s *ipmodel.IPPoolV1Status) {
		s.EntryCount = int64(len(entries))
		s.UpdatedAt = time.Now()
		s.Checksum = hex.EncodeToString(hf.Sum(nil))
	})
	if err == nil {
		notifyIPPoolChanged(ctx, groupID)
	}

	actionName := "ManagePoolEntry"
	commonaudit.FromContext(ctx).Log(actionName, targetPrefix.String(), mode, err == nil)

	return err
}

func (s *IPPoolService) PreviewPool(ctx context.Context, groupID string, cursorStr string, limit int, search string) (*ipmodel.IPPoolPreviewResponse, error) {
	cursor, _ := strconv.ParseInt(cursorStr, 10, 64)
	if err := requireIPResourceOrGlobal(ctx, ipGroupResource(groupID)); err != nil {
		return nil, err
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

	res := &ipmodel.IPPoolPreviewResponse{
		Total:   int64(reader.EntryCount()),
		Entries: make([]ipmodel.IPPoolEntry, 0),
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

		common.SortTags(tags)
		res.Entries = append(res.Entries, ipmodel.IPPoolEntry{
			CIDR: prefix.String(),
			Tags: tags,
		})
		matched++
	}

	// 获取当前文件偏移量作为下一个 cursor
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

func notifyIPPoolChanged(ctx context.Context, groupID string) {
	common.UpdateGlobalVersion(ctx, "network/ip/pool/"+groupID)
	common.NotifyCluster(ctx, common.EventIPPoolChanged, groupID)
}
