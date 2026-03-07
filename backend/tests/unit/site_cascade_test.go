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

func TestSitePoolCascadeDeleteAndDependencies(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()

	engine := site.NewAnalysisEngine(nil)
	service := site.NewSitePoolService(engine)

	// 1. Create a Pool
	group := &models.SiteGroup{ID: "test_site_pool", Name: "Test Site Pool"}
	err := service.CreateGroup(ctx, group)
	assert.NoError(t, err)

	// Create dummy data file
	_ = afero.WriteFile(common.FS, "network/site/pools/test_site_pool.bin", []byte("dummy"), 0644)

	// 2. Test Export Dependency
	export := &models.SiteExport{Name: "Dep Site Export", Rule: "true", GroupIDs: []string{"test_site_pool"}}
	_ = service.CreateExport(ctx, export)

	err = service.DeleteGroup(ctx, "test_site_pool")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "referenced by export")

	// Delete export to clear dependency
	_ = service.DeleteExport(ctx, export.ID)

	// 3. Test Successful Deletion and Cache Clearing
	// Fill cache
	_, _ = engine.GetMatcher(ctx, "test_site_pool")

	err = service.DeleteGroup(ctx, "test_site_pool")
	assert.NoError(t, err)

	// Verify file deleted
	exists, _ := afero.Exists(common.FS, "network/site/pools/test_site_pool.bin")
	assert.False(t, exists)

	// Verify group deleted from DB
	_, err = service.GetGroup(ctx, "test_site_pool")
	assert.Error(t, err)
}
