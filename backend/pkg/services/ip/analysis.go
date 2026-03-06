package ip

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
	"io"
	"net/netip"
	"path/filepath"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
)

type AnalysisEngine struct {
	mu         sync.RWMutex
	trieCache  *lru.Cache[string, *IPPoolTrie]
	mmdb       *MMDBManager
}

func NewAnalysisEngine(mmdb *MMDBManager) *AnalysisEngine {
	cache, _ := lru.New[string, *IPPoolTrie](32) // 缓存 32 个池的 Trie
	return &AnalysisEngine{
		trieCache: cache,
		mmdb:      mmdb,
	}
}

// GetTrie 获取或构建指定池的 Trie
func (e *AnalysisEngine) GetTrie(ctx context.Context, groupID string) (*IPPoolTrie, error) {
	// 检查缓存
	if val, ok := e.trieCache.Get(groupID); ok {
		return val, nil
	}

	// 锁定构建过程 (可以使用更细粒度的锁，但这里为了简单先用全局)
	e.mu.Lock()
	defer e.mu.Unlock()

	// Double check
	if val, ok := e.trieCache.Get(groupID); ok {
		return val, nil
	}

	// 从 VFS 加载并构建
	poolPath := filepath.Join(PoolsDir, groupID+".bin")
	f, err := common.FS.Open(poolPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open pool data: %w", err)
	}
	defer f.Close()

	reader, err := NewReader(f)
	if err != nil {
		return nil, fmt.Errorf("failed to create reader: %w", err)
	}

	trie := NewIPPoolTrie()
	for {
		prefix, tags, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		trie.Insert(prefix, tags)
	}

	e.trieCache.Add(groupID, trie)
	return trie, nil
}

// HitTest 研判 API
func (e *AnalysisEngine) HitTest(ctx context.Context, ipStr string, groupIDs []string) (*models.IPAnalysisResult, error) {
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil, err
	}

	// 如果没有指定 groupIDs，则查询所有
	if len(groupIDs) == 0 {
		groups, _, err := repo.ListGroups(ctx, 1, 1000, "")
		if err != nil {
			return nil, err
		}
		for _, g := range groups {
			groupIDs = append(groupIDs, g.ID)
		}
	}

	for _, gid := range groupIDs {
		trie, err := e.GetTrie(ctx, gid)
		if err != nil {
			continue
		}
		prefix, tags, ok := trie.Lookup(ip)
		if ok {
			return &models.IPAnalysisResult{
				Matched: true,
				CIDR:    prefix.String(),
				Tags:    tags,
			}, nil
		}
	}

	return &models.IPAnalysisResult{Matched: false}, nil
}

// Info 情报查询 API
func (e *AnalysisEngine) Info(ctx context.Context, ipStr string) (*models.IPInfoResponse, error) {
	return e.mmdb.Lookup(ipStr)
}
