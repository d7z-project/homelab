package ip_test

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/ip"
	"homelab/tests"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIPPoolCascadeDeleteAndDependencies(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()

	mmdbManager := ip.NewMMDBManager(nil)
	analysisEngine := ip.NewAnalysisEngine(mmdbManager)
	exportManager := ip.NewExportManager(analysisEngine)
	service := ip.NewIPPoolService(analysisEngine, exportManager)

	// 1. Create a Pool
	group := &models.IPPool{ID: "test_pool", Meta: models.IPPoolV1Meta{Name: "Test Pool"}}
	err := service.CreateGroup(ctx, group)
	assert.NoError(t, err)

	// Create dummy data file
	_ = afero.WriteFile(common.FS, "network/ip/pools/test_pool.bin", []byte("dummy"), 0644)

	// 2. Test Export Dependency
	export := &models.IPExport{Meta: models.IPExportV1Meta{Name: "Dep Export", Rule: "true", GroupIDs: []string{"test_pool"}}}
	_ = service.CreateExport(ctx, export)

	err = service.DeleteGroup(ctx, "test_pool")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "referenced by export")

	// Delete export to clear dependency
	_ = service.DeleteExport(ctx, export.ID)

	// 3. Test Sync Policy Dependency
	policy := &models.IPSyncPolicy{Meta: models.IPSyncPolicyV1Meta{Name: "Dep Policy", TargetGroupID: "test_pool", Enabled: true, SourceURL: "http://example.com"}}
	_ = service.CreateSyncPolicy(ctx, policy)

	err = service.DeleteGroup(ctx, "test_pool")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "referenced by sync policy")

	// Delete policy to clear dependency
	_ = service.DeleteSyncPolicy(ctx, policy.ID)

	// 4. Test Successful Deletion and Cache Clearing
	// Fill cache
	_, _ = analysisEngine.GetTrie(ctx, "test_pool")

	err = service.DeleteGroup(ctx, "test_pool")
	assert.NoError(t, err)

	// Verify file deleted
	exists, _ := afero.Exists(common.FS, "network/ip/pools/test_pool.bin")
	assert.False(t, exists)

	// Verify group deleted from DB
	_, err = service.GetGroup(ctx, "test_pool")
	assert.Error(t, err)
}
