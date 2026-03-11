package site_test

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/site"
	"homelab/tests"
	"fmt"
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
	service := site.NewSitePoolService(engine, nil)

	// 1. Create a Pool
	group := &models.SiteGroup{ID: "test_site_pool", Meta: models.SiteGroupV1Meta{Name: "Test Site Pool"}}
	err := service.CreateGroup(ctx, group)
	assert.NoError(t, err)

	// Create dummy data file
	_ = afero.WriteFile(common.FS, fmt.Sprintf("network/site/pools/%s.bin", group.ID), []byte("dummy"), 0644)

	// 2. Test Export Dependency
	export := &models.SiteExport{Meta: models.SiteExportV1Meta{Name: "Dep Site Export", Rule: "true", GroupIDs: []string{group.ID}}}
	_ = service.CreateExport(ctx, export)

	err = service.DeleteGroup(ctx, group.ID)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "referenced by export")

	// Delete export to clear dependency
	_ = service.DeleteExport(ctx, export.ID)

	// 3. Test Successful Deletion and Cache Clearing
	// Fill cache
	_, _ = engine.GetMatcher(ctx, group.ID)

	err = service.DeleteGroup(ctx, group.ID)
	assert.NoError(t, err)

	// Verify file deleted
	exists, _ := afero.Exists(common.FS, fmt.Sprintf("network/site/pools/%s.bin", group.ID))
	assert.False(t, exists)

	// Verify group deleted from DB
	_, err = service.GetGroup(ctx, group.ID)
	assert.Error(t, err)
}
