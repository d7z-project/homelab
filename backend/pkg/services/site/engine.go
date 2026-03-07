package site

import (
	"regexp"
	"strings"
)

type trieNode struct {
	children map[string]*trieNode
	isFull   bool
	isDomain bool
	tags     map[string]struct{}
}

func newTrieNode() *trieNode {
	return &trieNode{
		children: make(map[string]*trieNode),
		tags:     make(map[string]struct{}),
	}
}

// SuffixTrie 用于处理 Domain 和 Full 匹配
type SuffixTrie struct {
	root *trieNode
}

func NewSuffixTrie() *SuffixTrie {
	return &SuffixTrie{root: newTrieNode()}
}

func (t *SuffixTrie) Insert(ruleType uint8, value string, tags []string) {
	parts := strings.Split(value, ".")
	curr := t.root
	// 倒序插入
	for i := len(parts) - 1; i >= 0; i-- {
		p := parts[i]
		if _, ok := curr.children[p]; !ok {
			curr.children[p] = newTrieNode()
		}
		curr = curr.children[p]
	}
	if ruleType == 2 { // Domain
		curr.isDomain = true
	} else { // Full
		curr.isFull = true
	}
	for _, tag := range tags {
		curr.tags[tag] = struct{}{}
	}
}

func (t *SuffixTrie) Match(domain string) (matched bool, matchedPattern string, tags []string) {
	parts := strings.Split(domain, ".")
	curr := t.root

	var lastDomainNode *trieNode
	var lastDomainPattern string

	for i := len(parts) - 1; i >= 0; i-- {
		p := parts[i]
		if next, ok := curr.children[p]; ok {
			curr = next
			if curr.isDomain {
				lastDomainNode = curr
				lastDomainPattern = strings.Join(parts[i:], ".")
			}
		} else {
			// 不再匹配
			break
		}
	}

	// 优先 Full 匹配
	if curr.isFull && len(strings.Split(domain, ".")) == len(parts) { // 路径到底了
		return true, domain, getTags(curr.tags)
	}

	// 降级到最近的 Domain 匹配
	if lastDomainNode != nil {
		return true, lastDomainPattern, getTags(lastDomainNode.tags)
	}

	return false, "", nil
}

func getTags(m map[string]struct{}) []string {
	res := make([]string, 0, len(m))
	for k := range m {
		res = append(res, k)
	}
	return res
}

// KeywordMatcher 简单实现，AC 自动机在大规模 keyword 时更佳
type KeywordMatcher struct {
	rules map[string][]string // keyword -> tags
}

func NewKeywordMatcher() *KeywordMatcher {
	return &KeywordMatcher{rules: make(map[string][]string)}
}

func (m *KeywordMatcher) Insert(value string, tags []string) {
	m.rules[value] = append(m.rules[value], tags...)
}

func (m *KeywordMatcher) Match(domain string) (bool, string, []string) {
	var allTags []string
	var firstPat string
	matched := false
	for kw, tags := range m.rules {
		if strings.Contains(domain, kw) {
			if !matched {
				firstPat = kw
				matched = true
			}
			allTags = append(allTags, tags...)
		}
	}
	return matched, firstPat, allTags
}

// RegexMatcher
type RegexMatcher struct {
	rules []struct {
		re   *regexp.Regexp
		tags []string
	}
}

func NewRegexMatcher() *RegexMatcher {
	return &RegexMatcher{}
}

func (m *RegexMatcher) Insert(value string, tags []string) error {
	re, err := regexp.Compile(value)
	if err != nil {
		return err
	}
	m.rules = append(m.rules, struct {
		re   *regexp.Regexp
		tags []string
	}{re, tags})
	return nil
}

func (m *RegexMatcher) Match(domain string) (bool, string, []string) {
	var allTags []string
	var firstPat string
	matched := false
	for _, r := range m.rules {
		if r.re.MatchString(domain) {
			if !matched {
				firstPat = r.re.String()
				matched = true
			}
			allTags = append(allTags, r.tags...)
		}
	}
	return matched, firstPat, allTags
}

// CompositeMatcher 复合匹配引擎
type CompositeMatcher struct {
	trie    *SuffixTrie
	keyword *KeywordMatcher
	regex   *RegexMatcher
}

func NewCompositeMatcher() *CompositeMatcher {
	return &CompositeMatcher{
		trie:    NewSuffixTrie(),
		keyword: NewKeywordMatcher(),
		regex:   NewRegexMatcher(),
	}
}

func (m *CompositeMatcher) Match(domain string) (bool, uint8, string, []string) {
	var allTags []string
	var finalRuleType uint8
	var finalPattern string
	matched := false

	// 1. Trie (Domain/Full)
	if ok, pattern, tags := m.trie.Match(domain); ok {
		matched = true
		finalRuleType = 2 // Domain (Simplified)
		finalPattern = pattern
		allTags = append(allTags, tags...)
	}
	// 2. Keyword
	if ok, pattern, tags := m.keyword.Match(domain); ok {
		if !matched {
			finalRuleType = 0
			finalPattern = pattern
			matched = true
		}
		allTags = append(allTags, tags...)
	}
	// 3. Regex
	if ok, pattern, tags := m.regex.Match(domain); ok {
		if !matched {
			finalRuleType = 1
			finalPattern = pattern
			matched = true
		}
		allTags = append(allTags, tags...)
	}

	if !matched {
		return false, 0, "", nil
	}

	// Unique tags
	tagMap := make(map[string]struct{})
	var uniqueTags []string
	for _, t := range allTags {
		if _, ok := tagMap[t]; !ok {
			tagMap[t] = struct{}{}
			uniqueTags = append(uniqueTags, t)
		}
	}

	return true, finalRuleType, finalPattern, uniqueTags
}
