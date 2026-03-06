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

	lru "github.com/hashicorp/golang-lru/v2"
)

type AnalysisEngine struct {
	mu    sync.RWMutex
	cache *lru.Cache[string, *CompositeMatcher]
	mmdb  *ip.MMDBManager
}

func NewAnalysisEngine(mmdb *ip.MMDBManager) *AnalysisEngine {
	cache, _ := lru.New[string, *CompositeMatcher](32)
	return &AnalysisEngine{cache: cache, mmdb: mmdb}
}

func (e *AnalysisEngine) GetMatcher(ctx context.Context, groupID string) (*CompositeMatcher, error) {
	if val, ok := e.cache.Get(groupID); ok {
		return val, nil
	}

	e.mu.Lock()
	defer e.mu.Unlock()

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
		if err == io.EOF { break }
		if err != nil { return nil, err }

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

func (e *AnalysisEngine) HitTest(ctx context.Context, domain string, groupIDs []string) (*models.SiteAnalysisResult, error) {
	res := &models.SiteAnalysisResult{Matched: false}

	if len(groupIDs) == 0 {
		groups, _, err := repo.ListGroups(ctx, 1, 1000, "")
		if err == nil {
			for _, g := range groups {
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
			res.Matched = true
			res.RuleType = ruleType
			res.Pattern = pattern
			res.Tags = tags
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
