package site

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/site"
	"homelab/pkg/services/ip"
	"io"
	"net"
	"path/filepath"
	"strings"
	"sync"
	"time"

	lru "github.com/hashicorp/golang-lru/v2"
)

type AnalysisEngine struct {
	mu    sync.RWMutex
	cache *lru.Cache[string, *CompositeMatcher]
	mmdb  *ip.MMDBManager
}

func (e *AnalysisEngine) lockPool(ctx context.Context, id string) (func(), error) {
	lockKey := "network:site:matcher:build:" + id
	for {
		release := common.Locker.TryLock(ctx, lockKey)
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

func NewAnalysisEngine(mmdb *ip.MMDBManager) *AnalysisEngine {
	cache, _ := lru.New[string, *CompositeMatcher](32)
	engine := &AnalysisEngine{cache: cache, mmdb: mmdb}

	common.RegisterEventHandler(common.EventSitePoolChanged, func(ctx context.Context, payload string) {
		groupID := payload
		engine.RemoveCache(groupID)
	})

	return engine
}

func (e *AnalysisEngine) GetMatcher(ctx context.Context, groupID string) (*CompositeMatcher, error) {
	if val, ok := e.cache.Get(groupID); ok {
		return val, nil
	}

	// 1. 本地重入锁
	e.mu.Lock()
	if val, ok := e.cache.Get(groupID); ok {
		e.mu.Unlock()
		return val, nil
	}
	e.mu.Unlock()

	// 2. 分布式排队锁
	release, err := e.lockPool(ctx, groupID)
	if err != nil {
		return nil, err
	}
	defer release()

	if val, ok := e.cache.Get(groupID); ok {
		return val, nil
	}

	poolPath := filepath.Join(PoolsDir, groupID+".bin")
	f, err := common.FS.Open(poolPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open pool data: %w", err)
	}
	defer f.Close()

	reader, err := NewReader(f)
	if err != nil {
		return nil, err
	}

	matcher := NewCompositeMatcher()
	for {
		entry, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch entry.Type {
		case 0, 3: // Keyword or Full (Full handled by Trie too for simplicity in this implementation)
			matcher.trie.Insert(entry.Type, entry.Value, entry.Tags)
			if entry.Type == 0 {
				matcher.keyword.Insert(entry.Value, entry.Tags)
			}
		case 2: // Domain
			matcher.trie.Insert(entry.Type, entry.Value, entry.Tags)
		case 1: // Regex
			_ = matcher.regex.Insert(entry.Value, entry.Tags)
		}
	}

	e.cache.Add(groupID, matcher)
	return matcher, nil
}

func (e *AnalysisEngine) RemoveCache(groupID string) {
	e.cache.Remove(groupID)
}

func (e *AnalysisEngine) HitTest(ctx context.Context, domain string, groupIDs []string) (*models.SiteAnalysisResult, error) {
	res := &models.SiteAnalysisResult{Matched: false}

	if len(groupIDs) == 0 {
		resp, err := repo.ScanGroups(ctx, "", 1000, "")
		if err == nil {
			for _, g := range resp.Items {
				groupIDs = append(groupIDs, g.ID)
			}
		}
	}

	for _, gid := range groupIDs {
		matcher, err := e.GetMatcher(ctx, gid)
		if err != nil {
			continue
		}

		if ok, ruleType, pattern, tags := matcher.Match(domain); ok {
			// 1. 获取池名称
			poolName := gid
			if group, err := repo.GetGroup(ctx, gid); err == nil && group.Name != "" {
				poolName = group.Name
			}

			// 2. 处理标签：去重并转换内部 ID
			finalTags := make([]string, 0, len(tags)+1)
			finalTags = append(finalTags, "地址池: "+poolName)
			tagSet := make(map[string]struct{})
			tagSet[poolName] = struct{}{}

			for _, t := range tags {
				// 强制小写处理，确保匹配稳健
				tid := strings.ToLower(t)
				displayTag := t
				if strings.HasPrefix(tid, "_") {
					// 尝试作为同步策略查找
					if policy, err := repo.GetSyncPolicy(ctx, tid); err == nil && policy.Name != "" {
						displayTag = "策略: " + policy.Name
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
			res.Matched = true
			res.RuleType = ruleType
			res.Pattern = pattern
			res.Tags = finalTags
			break
		}
	}

	// Always attempt DNS analysis for domain intelligence
	res.DNS = e.dnsLookup(domain)

	return res, nil
}

func (e *AnalysisEngine) dnsLookup(domain string) *models.SiteDNSAnalysis {
	res := &models.SiteDNSAnalysis{}

	// 1. A & AAAA Records
	ips, err := net.LookupIP(domain)
	if err == nil {
		for _, ip := range ips {
			ipStr := ip.String()
			var info *models.IPInfoResponse
			if e.mmdb != nil {
				info, _ = e.mmdb.Lookup(ipStr)
			}
			if info == nil {
				info = &models.IPInfoResponse{IP: ipStr}
			}

			if ip.To4() != nil {
				res.A = append(res.A, *info)
			} else {
				res.AAAA = append(res.AAAA, *info)
			}
		}
	}

	// 2. CNAME Records
	cname, err := net.LookupCNAME(domain)
	if err == nil && strings.TrimRight(cname, ".") != strings.TrimRight(domain, ".") {
		res.CNAME = append(res.CNAME, cname)
	}

	// 3. SOA Records (using LookupNS as proxy for authority)
	ns, err := net.LookupNS(domain)
	if err == nil {
		for _, n := range ns {
			res.SOA = append(res.SOA, n.Host)
		}
	}

	return res
}
