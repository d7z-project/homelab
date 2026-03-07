package unit

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	"homelab/pkg/services/ip"
	"homelab/tests"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIPSyncLogic(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	common.FS = afero.NewMemMapFs()
	mmdb := ip.NewMMDBManager()
	service := ip.NewIPPoolService(mmdb)
	ctx := commonauth.WithPermissions(context.Background(), &models.ResourcePermissions{AllowedAll: true})

	group := &models.IPGroup{ID: "test_pool", Name: "Test Pool"}
	_ = service.CreateGroup(ctx, group)

	t.Run("Precision Overwrite and Multi-Source Coexistence", func(t *testing.T) {
		// Policy 1: 1.1.1.1, 1.1.1.2
		server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "1.1.1.1/32")
			fmt.Fprintln(w, "1.1.1.2/32")
		}))
		defer server1.Close()

		policy1 := &models.IPSyncPolicy{
			ID: "_p1", Name: "P1", SourceURL: server1.URL, Format: "text", Mode: "overwrite", TargetGroupID: "test_pool",
			Config: map[string]string{"allowPrivate": "true"},
		}
		_ = service.CreateSyncPolicy(ctx, policy1)
		_ = service.Sync(ctx, "_p1")

		// Policy 2: 2.2.2.2
		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "2.2.2.2/32")
		}))
		defer server2.Close()

		policy2 := &models.IPSyncPolicy{
			ID: "_p2", Name: "P2", SourceURL: server2.URL, Format: "text", Mode: "overwrite", TargetGroupID: "test_pool",
			Config: map[string]string{"allowPrivate": "true"},
		}
		_ = service.CreateSyncPolicy(ctx, policy2)
		_ = service.Sync(ctx, "_p2")

		// Verify Total: 3
		res, _ := service.PreviewPool(ctx, "test_pool", 0, 10, "")
		assert.Equal(t, int64(3), res.Total)

		// Update Policy 1: 1.1.1.1 -> 1.1.1.100 (overwrite old P1 records)
		server1_v2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "1.1.1.100/32")
		}))
		defer server1_v2.Close()
		policy1.SourceURL = server1_v2.URL
		_ = service.UpdateSyncPolicy(ctx, policy1)
		_ = service.Sync(ctx, "_p1")

		// Final Result: 1.1.1.100 (from P1) + 2.2.2.2 (from P2) = 2 records
		res, _ = service.PreviewPool(ctx, "test_pool", 0, 10, "")
		assert.Equal(t, int64(2), res.Total)

		cidrs := []string{}
		for _, e := range res.Entries {
			cidrs = append(cidrs, e.CIDR)
		}
		assert.Contains(t, cidrs, "1.1.1.100/32")
		assert.Contains(t, cidrs, "2.2.2.2/32")
		assert.NotContains(t, cidrs, "1.1.1.1/32")
	})

	t.Run("Aggregation and Tag Merging", func(t *testing.T) {
		// Same CIDR from different sources should be merged
		serverAgg := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "8.8.8.8/32")
		}))
		defer serverAgg.Close()

		policyA := &models.IPSyncPolicy{
			ID: "_pa", Name: "PA", SourceURL: serverAgg.URL, Format: "text", TargetGroupID: "test_pool",
			Config: map[string]string{"tags": "SOURCE_A", "allowPrivate": "true"},
		}
		_ = service.CreateSyncPolicy(ctx, policyA)
		_ = service.Sync(ctx, "_pa")

		policyB := &models.IPSyncPolicy{
			ID: "_pb", Name: "PB", SourceURL: serverAgg.URL, Format: "text", TargetGroupID: "test_pool",
			Config: map[string]string{"tags": "SOURCE_B", "allowPrivate": "true"},
		}
		_ = service.CreateSyncPolicy(ctx, policyB)
		_ = service.Sync(ctx, "_pb")

		// Total should be previous 2 + current 1 = 3
		res, _ := service.PreviewPool(ctx, "test_pool", 0, 100, "")
		assert.Equal(t, int64(3), res.Total)

		for _, e := range res.Entries {
			if e.CIDR == "8.8.8.8/32" {
				// Should have tags from both policies
				assert.ElementsMatch(t, []string{"_pa", "_pb", "source_a", "source_b"}, e.Tags)
			}
		}
	})

	t.Run("Append Mode with Deduplication", func(t *testing.T) {
		serverApp := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "9.9.9.9/32")
		}))
		defer serverApp.Close()

		policyApp := &models.IPSyncPolicy{
			ID: "_papp", Name: "PApp", SourceURL: serverApp.URL, Format: "text", Mode: "append", TargetGroupID: "test_pool",
			Config: map[string]string{"allowPrivate": "true"},
		}
		_ = service.CreateSyncPolicy(ctx, policyApp)

		_ = service.Sync(ctx, "_papp")
		_ = service.Sync(ctx, "_papp") // Sync again with same content

		// Total should be previous 3 + current 1 = 4 (not 5, because of deduplication)
		res, _ := service.PreviewPool(ctx, "test_pool", 0, 100, "")
		assert.Equal(t, int64(4), res.Total)
	})

	t.Run("Tag Removal During Sync Test", func(t *testing.T) {
		// 1. 初始状态：1.2.3.4 带上 TAG_A
		serverRem := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "1.2.3.4/32")
		}))
		defer serverRem.Close()

		policyRem := &models.IPSyncPolicy{
			ID: "_prem", Name: "Removal Test", SourceURL: serverRem.URL, Format: "text",
			Mode: "overwrite", TargetGroupID: "test_pool", Config: map[string]string{"tags": "tag_a", "allowPrivate": "true"},
		}
		_ = service.CreateSyncPolicy(ctx, policyRem)
		_ = service.Sync(ctx, "_prem")

		res, _ := service.PreviewPool(ctx, "test_pool", 0, 10, "1.2.3.4")
		assert.Contains(t, res.Entries[0].Tags, "tag_a")

		// 2. 模拟源数据更新：修改配置，将标签改为 tag_b
		policyRem.Config["tags"] = "tag_b"
		_ = service.UpdateSyncPolicy(ctx, policyRem)
		_ = service.Sync(ctx, "_prem")

		// 验证：tag_a 应该消失，只有 tag_b
		res, _ = service.PreviewPool(ctx, "test_pool", 0, 10, "1.2.3.4")
		assert.Contains(t, res.Entries[0].Tags, "tag_b")
		assert.NotContains(t, res.Entries[0].Tags, "tag_a")
	})
}
