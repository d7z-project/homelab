package unit

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/site"
	"homelab/tests"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestSiteTrie(t *testing.T) {
	trie := site.NewSuffixTrie()
	tags := []string{"cn", "direct"}
	trie.Insert(2, "google.com", tags)                 // Domain
	trie.Insert(3, "www.baidu.com", []string{"baidu"}) // Full

	// 1. Precise Match (Domain)
	ok, pat, resTags := trie.Match("mail.google.com")
	assert.True(t, ok)
	assert.Equal(t, "google.com", pat)
	assert.ElementsMatch(t, tags, resTags)

	// 2. Precise Match (Full)
	ok, pat, resTags = trie.Match("www.baidu.com")
	assert.True(t, ok)
	assert.Equal(t, "www.baidu.com", pat)
	assert.Contains(t, resTags, "baidu")

	// 3. Miss
	ok, _, _ = trie.Match("bing.com")
	assert.False(t, ok)
}

func TestSiteCodec(t *testing.T) {
	common.FS = afero.NewMemMapFs()
	codec := site.NewCodec()
	tags := []string{"tag1", "tag2"}
	entries := []site.Entry{
		{Type: 2, Value: "a.com", TagIndices: []uint32{0}},
		{Type: 3, Value: "b.com", TagIndices: []uint32{1}},
	}

	f, _ := common.FS.Create("test.bin")
	err := codec.WritePool(f, tags, entries)
	assert.NoError(t, err)
	f.Close()

	rf, _ := common.FS.Open("test.bin")
	reader, err := site.NewReader(rf)
	assert.NoError(t, err)
	assert.Equal(t, uint32(2), reader.EntryCount())

	e1, _ := reader.Next()
	assert.Equal(t, "a.com", e1.Value)
	assert.Contains(t, e1.Tags, "tag1")
	rf.Close()
}

func TestSiteService(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()

	engine := site.NewAnalysisEngine(nil)
	service := site.NewSitePoolService(engine)

	group := &models.SiteGroup{Name: "Test Site Pool"}
	err := service.CreateGroup(ctx, group)
	assert.NoError(t, err)
	assert.NotEmpty(t, group.ID)

	// Manage Entry
	req := &models.SitePoolEntryRequest{
		Type:  2,
		Value: "example.com",
		Tags:  []string{"test"},
	}
	err = service.ManagePoolEntry(ctx, group.ID, req, "add")
	assert.NoError(t, err)

	// Manage Entry with Uppercase Tags
	req = &models.SitePoolEntryRequest{
		Type:  2,
		Value: "uppercase.com",
		Tags:  []string{"  UPPER  "},
	}
	_ = req.Bind(nil) // Normalize tags to lowercase
	err = service.ManagePoolEntry(ctx, group.ID, req, "add")
	assert.NoError(t, err)

	// Preview and verify lowercase
	res2, _ := service.PreviewPool(ctx, group.ID, 0, 10, "UPPER")
	assert.Len(t, res2.Entries, 1)
	assert.Equal(t, "uppercase.com", res2.Entries[0].Value)
	assert.Equal(t, "upper", res2.Entries[0].Tags[0])
}
