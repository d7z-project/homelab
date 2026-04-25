package site

import (
	"regexp"
	"strings"
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
)

// --- Aho-Corasick for Keyword Matching ---

type acNode struct {
	children map[rune]*acNode
	fail     *acNode
	patterns []string
	tags     []string
}

func newACNode() *acNode {
	return &acNode{
		children: make(map[rune]*acNode),
	}
}

type KeywordMatcher struct {
	root          *acNode
	built         bool
	mu            sync.RWMutex
	patternToTags map[string][]string
}

func NewKeywordMatcher() *KeywordMatcher {
	return &KeywordMatcher{
		root:          newACNode(),
		patternToTags: make(map[string][]string),
	}
}

func (m *KeywordMatcher) Insert(value string, tags []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.built = false
	m.patternToTags[value] = append(m.patternToTags[value], tags...)
}

func (m *KeywordMatcher) Build() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.built {
		return
	}

	newRoot := newACNode()
	for pat, tags := range m.patternToTags {
		curr := newRoot
		for _, r := range pat {
			if _, ok := curr.children[r]; !ok {
				curr.children[r] = newACNode()
			}
			curr = curr.children[r]
		}
		curr.patterns = append(curr.patterns, pat)
		curr.tags = append(curr.tags, tags...)
	}

	// BFS to build failure links
	queue := make([]*acNode, 0)
	for _, child := range newRoot.children {
		child.fail = newRoot
		queue = append(queue, child)
	}

	for len(queue) > 0 {
		u := queue[0]
		queue = queue[1:]

		for r, v := range u.children {
			f := u.fail
			for f != nil {
				if next, ok := f.children[r]; ok {
					v.fail = next
					break
				}
				f = f.fail
			}
			if v.fail == nil {
				v.fail = newRoot
			}
			// Merge tags and patterns from failure node to handle overlapping patterns
			v.patterns = append(v.patterns, v.fail.patterns...)
			v.tags = append(v.tags, v.fail.tags...)
			queue = append(queue, v)
		}
	}

	m.root = newRoot
	m.built = true
}

func (m *KeywordMatcher) Match(domain string) (bool, string, []string) {
	m.mu.RLock()
	if !m.built {
		m.mu.RUnlock()
		m.Build()
		m.mu.RLock()
	}
	defer m.mu.RUnlock()

	curr := m.root
	var allTags []string
	var firstPat string
	matched := false

	tagSet := make(map[string]struct{})

	for _, r := range domain {
		for {
			if next, ok := curr.children[r]; ok {
				curr = next
				break
			}
			if curr == m.root {
				break
			}
			curr = curr.fail
		}

		if len(curr.patterns) > 0 {
			matched = true
			if firstPat == "" {
				firstPat = curr.patterns[0]
			}
			for _, t := range curr.tags {
				if _, ok := tagSet[t]; !ok {
					tagSet[t] = struct{}{}
					allTags = append(allTags, t)
				}
			}
		}
	}

	return matched, firstPat, allTags
}

// --- Suffix Trie for Domain/Full Matching ---

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

type SuffixTrie struct {
	root *trieNode
}

func NewSuffixTrie() *SuffixTrie {
	return &SuffixTrie{root: newTrieNode()}
}

func (t *SuffixTrie) Insert(ruleType uint8, value string, tags []string) {
	parts := strings.Split(value, ".")
	curr := t.root
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

func (t *SuffixTrie) Match(domain string) (bool, string, []string) {
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
			break
		}
	}

	if curr.isFull && len(strings.Split(domain, ".")) == len(parts) {
		return true, domain, getTags(curr.tags)
	}

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

// --- Regex Matcher with Pre-compilation Cache ---

var regexGlobalCache *lru.Cache[string, *regexp.Regexp]

func init() {
	regexGlobalCache, _ = lru.New[string, *regexp.Regexp](2048)
}

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
	var re *regexp.Regexp
	if cached, ok := regexGlobalCache.Get(value); ok {
		re = cached
	} else {
		var err error
		re, err = regexp.Compile(value)
		if err != nil {
			return err
		}
		regexGlobalCache.Add(value, re)
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

// --- Composite Matcher ---

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
		finalRuleType = 2
		finalPattern = pattern
		allTags = append(allTags, tags...)
	}
	// 2. Keyword (AC)
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
