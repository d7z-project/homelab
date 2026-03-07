package ip_test

import (
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/ip"
	"homelab/tests"
	"io"
	"net/netip"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIPCodec(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	codec := ip.NewCodec()
	tags := []string{"tag1", "tag2", "tag3"}
	entries := []ip.Entry{
		{Prefix: netip.MustParsePrefix("192.168.1.0/24"), TagIndices: []uint32{0, 1}},
		{Prefix: netip.MustParsePrefix("10.0.0.1/32"), TagIndices: []uint32{1, 2}},
		{Prefix: netip.MustParsePrefix("2001:db8::/32"), TagIndices: []uint32{0}},
	}

	testFile := "test_codec.bin"
	f, err := common.FS.Create(testFile)
	assert.NoError(t, err)

	err = codec.WritePool(f, tags, entries)
	assert.NoError(t, err)
	f.Close()

	rf, err := common.FS.Open(testFile)
	assert.NoError(t, err)
	defer rf.Close()

	reader, err := ip.NewReader(rf)
	assert.NoError(t, err)

	assert.Equal(t, uint32(len(entries)), reader.EntryCount())
	assert.Equal(t, tags, reader.Tags())

	// Read entries back
	var readEntries []ip.Entry
	for {
		prefix, itemTags, err := reader.Next()
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)

		indices := make([]uint32, 0)
		for _, tag := range itemTags {
			for i, t := range tags {
				if t == tag {
					indices = append(indices, uint32(i))
				}
			}
		}
		readEntries = append(readEntries, ip.Entry{Prefix: prefix, TagIndices: indices})
	}

	assert.Equal(t, len(entries), len(readEntries))
	for i := range entries {
		assert.Equal(t, entries[i].Prefix.String(), readEntries[i].Prefix.String())
		assert.Equal(t, entries[i].TagIndices, readEntries[i].TagIndices)
	}
}

func TestIPTrie(t *testing.T) {
	trie := ip.NewIPPoolTrie()
	trie.Insert(netip.MustParsePrefix("192.168.1.0/24"), []string{"internal"})
	trie.Insert(netip.MustParsePrefix("192.168.1.100/32"), []string{"server"})
	trie.Insert(netip.MustParsePrefix("10.0.0.0/8"), []string{"private"})

	// 测试精确匹配
	p, tags, ok := trie.Lookup(netip.MustParseAddr("192.168.1.100"))
	assert.True(t, ok)
	assert.Equal(t, "192.168.1.100/32", p.String())
	assert.Contains(t, tags, "server")

	// 测试最长前缀匹配
	p, tags, ok = trie.Lookup(netip.MustParseAddr("192.168.1.50"))
	assert.True(t, ok)
	assert.Equal(t, "192.168.1.0/24", p.String())
	assert.Contains(t, tags, "internal")

	// 测试不匹配
	_, _, ok = trie.Lookup(netip.MustParseAddr("8.8.8.8"))
	assert.False(t, ok)

	// 测试 IPv6
	trie.Insert(netip.MustParsePrefix("2001:db8::/32"), []string{"ipv6"})
	p, tags, ok = trie.Lookup(netip.MustParseAddr("2001:db8:1234::1"))
	assert.True(t, ok)
	assert.Equal(t, "2001:db8::/32", p.String())
	assert.Contains(t, tags, "ipv6")
}

func TestIPService(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	mmdb := ip.NewMMDBManager()
	service := ip.NewIPPoolService(mmdb)

	// 1. 创建 Group
	group := &models.IPGroup{
		ID:   "test_group",
		Name: "Test Group",
	}
	err := service.CreateGroup(ctx, group)
	assert.NoError(t, err)

	// 2. 获取 Group
	g, err := service.GetGroup(ctx, "test_group")
	assert.NoError(t, err)
	assert.Equal(t, "Test Group", g.Name)

	// 3. 更新 Group
	g.Name = "New Name"
	err = service.UpdateGroup(ctx, g)
	assert.NoError(t, err)
	g2, _ := service.GetGroup(ctx, "test_group")
	assert.Equal(t, "New Name", g2.Name)

	// 4. 列表
	list, total, err := service.ListGroups(ctx, 1, 10, "")
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "test_group", list[0].ID)

	// 5. Lookup (Discovery)
	lg, err := service.LookupGroup(ctx, "test_group")
	assert.NoError(t, err)
	assert.NotNil(t, lg)

	// 6. 删除
	err = service.DeleteGroup(ctx, "test_group")
	assert.NoError(t, err)

	_, err = service.GetGroup(ctx, "test_group")
	assert.Error(t, err)
}

func TestIPPreviewCursor(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	common.FS = afero.NewMemMapFs()
	mmdb := ip.NewMMDBManager()
	service := ip.NewIPPoolService(mmdb)

	// 1. Create group and many entries
	group := &models.IPGroup{ID: "cursor_pool", Name: "Cursor Pool"}
	_ = service.CreateGroup(ctx, group)

	entries := make([]ip.Entry, 10)
	for i := 0; i < 10; i++ {
		addr := netip.MustParseAddr(fmt.Sprintf("1.1.1.%d", i))
		entries[i] = ip.Entry{
			Prefix:     netip.PrefixFrom(addr, 32),
			TagIndices: []uint32{},
		}
	}

	codec := ip.NewCodec()
	f, _ := common.FS.Create("network/ip/pools/cursor_pool.bin")
	_ = codec.WritePool(f, []string{}, entries)
	f.Close()

	// 2. Test Preview with limit
	res, err := service.PreviewPool(ctx, "cursor_pool", 0, 3, "")
	assert.NoError(t, err)
	assert.Len(t, res.Entries, 3)
	assert.True(t, res.NextCursor > 0)

	// 3. Use cursor to get next page
	res2, err := service.PreviewPool(ctx, "cursor_pool", res.NextCursor, 3, "")
	assert.NoError(t, err)
	assert.Len(t, res2.Entries, 3)
	assert.Equal(t, "1.1.1.3/32", res2.Entries[0].CIDR)
}

func TestIPIntelligence(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	mmdb := ip.NewMMDBManager()

	// 测试私有 IP
	res, err := mmdb.Lookup("192.168.1.1")
	assert.NoError(t, err)
	assert.Equal(t, uint(0), res.ASN)
	assert.Equal(t, "Private Network", res.Org)
	assert.Equal(t, "内网", res.Country)
	assert.Equal(t, "私有地址", res.City)
	assert.Equal(t, "0.000000,0.000000", res.Location)

	// 测试回环 IP
	res, err = mmdb.Lookup("127.0.0.1")
	assert.NoError(t, err)
	assert.Equal(t, "内网", res.Country)

	// 测试无效 IP
	_, err = mmdb.Lookup("invalid")
	assert.Error(t, err)
}
