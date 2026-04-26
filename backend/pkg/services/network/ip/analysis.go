package ip

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	networkcommon "homelab/pkg/models/network/common"
	ipmodel "homelab/pkg/models/network/ip"
	repo "homelab/pkg/repositories/network/ip"
	runtimepkg "homelab/pkg/runtime"
	ruleservice "homelab/pkg/services/rules"
	"io"
	"net/netip"
	"path/filepath"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

type AnalysisEngine struct {
	mu        sync.RWMutex
	trieCache *lru.Cache[string, *IPPoolTrie]
	enricher  ruleservice.IPEnricher
}

func (e *AnalysisEngine) lockPool(ctx context.Context, id string) (func(), error) {
	lockKey := "network:ip:trie:build:" + id
	for {
		release := runtimepkg.LockerFromContext(ctx).TryLock(ctx, lockKey)
		if release != nil {
			return release, nil
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func NewAnalysisEngine(enricher ruleservice.IPEnricher) *AnalysisEngine {
	cache, _ := lru.New[string, *IPPoolTrie](32) // 缓存 32 个池的 Trie
	engine := &AnalysisEngine{
		trieCache: cache,
		enricher:  enricher,
	}

	common.RegisterEventHandler(common.EventIPPoolChanged, func(ctx context.Context, payload common.ResourceEventPayload) {
		engine.RemoveCache(payload.ID)
	})

	return engine
}

// GetTrie 获取或构建指定池的 Trie
func (e *AnalysisEngine) GetTrie(ctx context.Context, groupID string) (*IPPoolTrie, error) {
	// 检查缓存
	if val, ok := e.trieCache.Get(groupID); ok {
		return val, nil
	}

	// 1. 本地重入锁/并发保护 (防止当前实例内部多协程同时触发)
	e.mu.Lock()
	if val, ok := e.trieCache.Get(groupID); ok {
		e.mu.Unlock()
		return val, nil
	}
	e.mu.Unlock()

	// 2. 分布式排队锁 (防止集群内所有实例同时读取存储并构建，序列化 I/O)
	release, err := e.lockPool(ctx, groupID)
	if err != nil {
		return nil, err
	}
	defer release()

	// 再次检查缓存 (可能上一个持有分布式锁的实例已经建好了，
	// 但当前实例还没建，所以这里其实还是要建一次。
	// 但至少现在我们保证同一时间只有一个实例在执行大文件的读取和计算。)
	if val, ok := e.trieCache.Get(groupID); ok {
		return val, nil
	}

	// 从 VFS 加载并构建
	poolPath := filepath.Join(PoolsDir, groupID+".bin")
	f, err := runtimepkg.FSFromContext(ctx).Open(poolPath)
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

func (e *AnalysisEngine) RemoveCache(groupID string) {
	e.trieCache.Remove(groupID)
}

// HitTest 研判 API
func (e *AnalysisEngine) HitTest(ctx context.Context, ipStr string, groupIDs []string) (*ipmodel.IPAnalysisResult, error) {
	ip, err := netip.ParseAddr(ipStr)
	if err != nil {
		return nil, err
	}

	// 如果没有指定 groupIDs，则查询所有
	if len(groupIDs) == 0 {
		groups, err := repo.ScanAllPools(ctx)
		if err != nil {
			return nil, err
		}
		for _, g := range groups {
			groupIDs = append(groupIDs, g.ID)
		}
	}

	var allMatches []ipmodel.IPAnalysisMatch
	for _, gid := range groupIDs {
		trie, err := e.GetTrie(ctx, gid)
		if err != nil {
			continue
		}
		prefix, tags, ok := trie.Lookup(ip)
		if ok {
			// 1. 获取池名称
			poolName := gid
			if group, err := repo.GetPool(ctx, gid); err == nil && group.Meta.Name != "" {
				poolName = group.Meta.Name
			}

			// 2. 处理标签：去重并转换内部 ID
			finalTags := make([]string, 0, len(tags)+1)
			finalTags = append(finalTags, "地址池: "+poolName) // 注入来源池名称
			tagSet := make(map[string]struct{})
			tagSet[poolName] = struct{}{}

			for _, t := range tags {
				// 强制小写处理，确保匹配稳健
				tid := strings.ToLower(t)
				displayTag := t
				if strings.HasPrefix(tid, "_") {
					// 尝试作为同步策略查找
					if policy, err := repo.GetSyncPolicy(ctx, tid); err == nil && policy.Meta.Name != "" {
						displayTag = "策略: " + policy.Meta.Name
					} else {
						// 如果无法解析为策略名称，则隐藏该内部标签
						continue
					}
				}

				if _, exists := tagSet[displayTag]; !exists {
					tagSet[displayTag] = struct{}{}
					finalTags = append(finalTags, displayTag)
				}
			}
			common.SortTags(finalTags)
			allMatches = append(allMatches, ipmodel.IPAnalysisMatch{
				CIDR: prefix.String(),
				Tags: finalTags,
			})
		}
	}

	if len(allMatches) > 0 {
		result := &ipmodel.IPAnalysisResult{
			Matched: true,
			Matches: allMatches,
		}
		return result, nil
	}

	return &ipmodel.IPAnalysisResult{Matched: false}, nil
}

// Info 情报查询 API
func (e *AnalysisEngine) Info(ctx context.Context, ipStr string) (*networkcommon.IPInfoResponse, error) {
	_ = ctx
	if e.enricher == nil {
		return &networkcommon.IPInfoResponse{IP: ipStr}, nil
	}
	return e.enricher.Lookup(ipStr)
}
