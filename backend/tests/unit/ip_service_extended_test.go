package unit

import (
	"homelab/pkg/models"
	"homelab/pkg/services/ip"
	"homelab/tests"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIPServiceExtended(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	mmdb := ip.NewMMDBManager()
	service := ip.NewIPPoolService(mmdb)

	// 1. Test UpdateGroup
	group := &models.IPGroup{ID: "update_group", Name: "Old Name"}
	_ = service.CreateGroup(ctx, group)

	group.Name = "New Name"
	err := service.UpdateGroup(ctx, group)
	assert.NoError(t, err)

	g, _ := service.GetGroup(ctx, "update_group")
	assert.Equal(t, "New Name", g.Name)

	// 2. Test PreviewExport (unimplemented logic usually returns empty or error, but let's see)
	export := &models.IPExport{Name: "Exp1", Rule: "true", GroupIDs: []string{group.ID}}
	_ = service.CreateExport(ctx, export)

	_, err = service.PreviewExport(ctx, &models.IPExportPreviewRequest{
		Rule:     export.Rule,
		GroupIDs: export.GroupIDs,
	})
	assert.NoError(t, err)

	// 3. Test LookupGroup / LookupExport (Discovery integration)
	lg, err := service.LookupGroup(ctx, "update_group")
	assert.NoError(t, err)
	assert.NotNil(t, lg)

	le, err := service.LookupExport(ctx, export.ID)
	assert.NoError(t, err)
	assert.NotNil(t, le)

	// 4. Test SyncPolicies retrieval
	policy := &models.IPSyncPolicy{ID: "policy1", Name: "Policy 1", SourceURL: "http://example.com/ip.txt"}
	_ = service.CreateSyncPolicy(ctx, policy)

	p, err := service.GetSyncPolicy(ctx, "policy1")
	assert.NoError(t, err)
	assert.Equal(t, "Policy 1", p.Name)

	policies, total, err := service.ListSyncPolicies(ctx, 1, 10, "")
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, policies, 1)

	// 5. Test SSRF in Sync
	ssrfPolicy := &models.IPSyncPolicy{ID: "ssrf", Name: "SSRF Policy", SourceURL: "http://192.168.1.1/data"}
	_ = service.CreateSyncPolicy(ctx, ssrfPolicy)

	err = service.Sync(ctx, "ssrf")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SSRF detected")
}

func TestIPValidationService(t *testing.T) {
	mmdb := ip.NewMMDBManager()

	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	service := ip.NewIPPoolService(mmdb)

	g := &models.IPGroup{ID: "entry_pool", Name: "Entry Pool"}
	_ = service.CreateGroup(ctx, g)
	_ = service.ManagePoolEntry(ctx, "entry_pool", &models.IPPoolEntryRequest{
		CIDR:    "1.1.1.1/32",
		NewTags: []string{"tag1"},
	}, "add")

	res, err := service.PreviewExport(ctx, &models.IPExportPreviewRequest{
		Rule: "true", GroupIDs: []string{"entry_pool"},
	})
	assert.NoError(t, err)
	assert.NotEmpty(t, res)
}
