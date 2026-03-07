package ip

import (
	"net/netip"
)

type trieNode struct {
	children [2]*trieNode
	prefix   *netip.Prefix
	tags     []string
}

// IPPoolTrie 实现了一个基于 IP 二进制位的基数树，用于前缀匹配
type IPPoolTrie struct {
	roots [2]*trieNode // 0: IPv4, 1: IPv6
}

func NewIPPoolTrie() *IPPoolTrie {
	return &IPPoolTrie{
		roots: [2]*trieNode{{}, {}},
	}
}

func (t *IPPoolTrie) Insert(p netip.Prefix, tags []string) {
	addr := p.Addr()
	var family int
	var ipBytes []byte
	if addr.Is4() {
		family = 0
		b4 := addr.As4()
		ipBytes = b4[:]
	} else {
		family = 1
		b16 := addr.As16()
		ipBytes = b16[:]
	}

	curr := t.roots[family]
	bits := p.Bits()

	for i := 0; i < bits; i++ {
		byteIdx := i / 8
		bitIdx := 7 - (i % 8)
		bit := (ipBytes[byteIdx] >> uint(bitIdx)) & 1

		if curr.children[bit] == nil {
			curr.children[bit] = &trieNode{}
		}
		curr = curr.children[bit]
	}

	// 如果已经有 tags 了，合并（或者根据业务逻辑替换，这里选择追加并去重）
	curr.prefix = &p
	curr.tags = append(curr.tags, tags...)
}

func (t *IPPoolTrie) Lookup(ip netip.Addr) (netip.Prefix, []string, bool) {
	var family int
	var ipBytes []byte
	var maxBits int
	if ip.Is4() {
		family = 0
		b4 := ip.As4()
		ipBytes = b4[:]
		maxBits = 32
	} else {
		family = 1
		b16 := ip.As16()
		ipBytes = b16[:]
		maxBits = 128
	}

	curr := t.roots[family]
	var lastMatchedPrefix netip.Prefix
	var lastMatchedTags []string
	var matched bool

	for i := 0; i < maxBits; i++ {
		if curr.prefix != nil {
			lastMatchedPrefix = *curr.prefix
			lastMatchedTags = curr.tags
			matched = true
		}

		byteIdx := i / 8
		bitIdx := 7 - (i % 8)
		bit := (ipBytes[byteIdx] >> uint(bitIdx)) & 1

		if curr.children[bit] == nil {
			break
		}
		curr = curr.children[bit]
	}

	// 检查最后一个节点（最长前缀匹配）
	if curr.prefix != nil {
		lastMatchedPrefix = *curr.prefix
		lastMatchedTags = curr.tags
		matched = true
	}

	return lastMatchedPrefix, lastMatchedTags, matched
}
