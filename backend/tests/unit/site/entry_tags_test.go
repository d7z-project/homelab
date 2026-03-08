package site_test

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/site"
	"homelab/tests"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestSiteEntryTagsGranularUpdate(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()

	service := site.NewSitePoolService(nil, nil)

	group := &models.SiteGroup{ID: "tag_test", Name: "Tag Test"}
	_ = service.CreateGroup(ctx, group)

	t.Run("Prevent manual creation of internal tags", func(t *testing.T) {
		req := &models.SitePoolEntryRequest{
			Type:    2,
			Value:   "google.com",
			NewTags: []string{"_internal"},
		}
		// Bind should fail
		err := req.Bind(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "reserved for internal use")
	})

	t.Run("Granular Tag Update and Internal Tag Preservation", func(t *testing.T) {
		// 1. Initial creation with an internal tag (simulated, since Bind normally blocks it)
		// We bypass Bind in the test by calling ManagePoolEntry directly if needed,
		// but ManagePoolEntry doesn't call Bind. Controllers do.
		// However, ManagePoolEntry does NOT prevent internal tags because they might come from system processors.

		// Setup an entry with an internal tag + a user tag
		// We can't use generic ManagePoolEntry to add internal tag if we want to simulate system-added ones?
		// Actually, ManagePoolEntry just takes what's in req.NewTags.

		// Simulate a system added an internal tag "_src_intel"
		// We'll use a trick: we'll call ManagePoolEntry with it, as it doesn't have the Bind check (Bind is on Model)
		_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{
			Type: 2, Value: "a.com", NewTags: []string{"_src_intel", "user_tag_1"},
		}, "add")

		preview, _ := service.PreviewPool(ctx, group.ID, 0, 10, "")
		assert.ElementsMatch(t, []string{"_src_intel", "user_tag_1"}, preview.Entries[0].Tags)

		// 2. Update user_tag_1 to user_tag_2 using OldTags/NewTags
		err := service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{
			Type:    2,
			Value:   "a.com",
			OldTags: []string{"user_tag_1"},
			NewTags: []string{"user_tag_2"},
		}, "update")
		assert.NoError(t, err)

		preview, _ = service.PreviewPool(ctx, group.ID, 0, 10, "")
		// Should have: _src_intel (preserved), user_tag_2 (added)
		// user_tag_1 should be gone because it was in OldTags
		assert.ElementsMatch(t, []string{"_src_intel", "user_tag_2"}, preview.Entries[0].Tags)

		// 3. Update without OldTags should replace all non-internal tags
		err = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{
			Type:    2,
			Value:   "a.com",
			NewTags: []string{"user_tag_3"},
		}, "update")
		assert.NoError(t, err)

		preview, _ = service.PreviewPool(ctx, group.ID, 0, 10, "")
		// Should have: _src_intel (preserved), user_tag_3 (fully replaced user_tag_2)
		assert.ElementsMatch(t, []string{"_src_intel", "user_tag_3"}, preview.Entries[0].Tags)
	})

	t.Run("Tag Sorting Logic", func(t *testing.T) {
		_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{
			Type: 2, Value: "sort.com", NewTags: []string{"z_tag", "_a_internal", "m_tag", "_z_internal"},
		}, "add")

		preview, _ := service.PreviewPool(ctx, group.ID, 0, 10, "sort.com")
		// The PreviewPool logic now sorts tags: internal first, then alphabetically
		// _a_internal, _z_internal, m_tag, z_tag
		assert.Equal(t, []string{"_a_internal", "_z_internal", "m_tag", "z_tag"}, preview.Entries[0].Tags)
	})
}
