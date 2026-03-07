package ip_test

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
	"time"

	"bytes"
	"net/netip"

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

	syncAndWait := func(id string) {
		_ = service.Sync(ctx, id)
		for i := 0; i < 50; i++ {
			p, _ := service.GetSyncPolicy(ctx, id)
			if p != nil && (p.LastStatus == "success" || p.LastStatus == "failed") {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
	}

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
		syncAndWait("_p1")

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
		syncAndWait("_p2")

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
		syncAndWait("_p1")

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
		syncAndWait("_pa")

		policyB := &models.IPSyncPolicy{
			ID: "_pb", Name: "PB", SourceURL: serverAgg.URL, Format: "text", TargetGroupID: "test_pool",
			Config: map[string]string{"tags": "SOURCE_B", "allowPrivate": "true"},
		}
		_ = service.CreateSyncPolicy(ctx, policyB)
		syncAndWait("_pb")

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

		syncAndWait("_papp")
		syncAndWait("_papp") // Sync again with same content

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
		syncAndWait("_prem")

		res, _ := service.PreviewPool(ctx, "test_pool", 0, 10, "1.2.3.4")
		assert.Contains(t, res.Entries[0].Tags, "tag_a")

		// 2. 模拟源数据更新：修改配置，将标签改为 tag_b
		policyRem.Config["tags"] = "tag_b"
		_ = service.UpdateSyncPolicy(ctx, policyRem)
		syncAndWait("_prem")

		// 验证：tag_a 应该消失，只有 tag_b
		res, _ = service.PreviewPool(ctx, "test_pool", 0, 10, "1.2.3.4")
		assert.Contains(t, res.Entries[0].Tags, "tag_b")
		assert.NotContains(t, res.Entries[0].Tags, "tag_a")
	})

	t.Run("Sync CSV and GeoIP Formats", func(t *testing.T) {
		// CSV
		serverCSV := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "ip,tag")
			fmt.Fprintln(w, "1.2.3.4,test-tag")
		}))
		defer serverCSV.Close()

		policyCSV := &models.IPSyncPolicy{
			ID: "csv_policy", Name: "CSV", SourceURL: serverCSV.URL, Format: "csv", TargetGroupID: "test_pool",
			Config: map[string]string{"allowPrivate": "true", "ipColumn": "0", "tagColumn": "1"},
		}
		_ = service.CreateSyncPolicy(ctx, policyCSV)
		syncAndWait("csv_policy")

		res, _ := service.PreviewPool(ctx, "test_pool", 0, 10, "1.2.3.4")
		assert.Contains(t, res.Entries[0].Tags, "test-tag")

		// GeoIP-DAT
		var buf bytes.Buffer
		groups := map[string][]netip.Prefix{"CN": {netip.MustParsePrefix("114.114.114.114/32")}}
		_ = ip.BuildV2RayGeoIP(&buf, groups)
		serverDat := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(buf.Bytes())
		}))
		defer serverDat.Close()

		policyDat := &models.IPSyncPolicy{
			ID: "v2ray_policy", Name: "V2Ray", SourceURL: serverDat.URL, Format: "geoip-dat", TargetGroupID: "test_pool",
			Config: map[string]string{"allowPrivate": "true", "code": "CN"},
		}
		_ = service.CreateSyncPolicy(ctx, policyDat)
		syncAndWait("v2ray_policy")

		res, _ = service.PreviewPool(ctx, "test_pool", 0, 10, "114.114.114.114")
		assert.Contains(t, res.Entries[0].Tags, "cn")
	})

	t.Run("SSRF Protection", func(t *testing.T) {
		ssrfPolicy := &models.IPSyncPolicy{
			ID: "ssrf", Name: "SSRF", SourceURL: "http://192.168.1.1/ips.txt",
			TargetGroupID: "test_pool", Format: "text",
		}
		_ = service.CreateSyncPolicy(ctx, ssrfPolicy)
		_ = service.Sync(ctx, "ssrf")

		// 稍微等等异步执行报错
		time.Sleep(200 * time.Millisecond)
		p, _ := service.GetSyncPolicy(ctx, "ssrf")
		assert.Equal(t, "failed", p.LastStatus)
		assert.Contains(t, p.ErrorMessage, "SSRF detected")
	})
}

func TestIPSyncFrameworkIntegration(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	mmdb := ip.NewMMDBManager()
	service := ip.NewIPPoolService(mmdb)

	t.Run("Reconcile Zombie Sync Task", func(t *testing.T) {
		policy := &models.IPSyncPolicy{ID: "zombie_p", Name: "Zombie"}
		_ = service.CreateSyncPolicy(ctx, policy)

		// 模拟一个假装正在跑的任务（通过 TaskManager 直接注入）
		task := &ip.SyncTask{ID: "zombie_p", Status: "Running", CreatedAt: time.Now()}
		service.GetSyncTasks().AddTask(task)

		// 执行自愈
		service.GetSyncTasks().Reconcile(ctx)

		retrieved, _ := service.GetSyncTasks().GetTask("zombie_p")
		assert.Equal(t, "Failed", retrieved.GetStatus())
		assert.Contains(t, retrieved.Error, "node failure")
	})

	t.Run("Cancellation of Sync Task", func(t *testing.T) {
		// 起一个一直在下载的 Mock HTTP 服务器
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-r.Context().Done():
					return
				case <-ticker.C:
					w.Write([]byte("data\n"))
				}
			}
		}))
		defer server.Close()

		group := &models.IPGroup{ID: "p_cancel", Name: "Cancel Pool"}
		_ = service.CreateGroup(ctx, group)

		policy := &models.IPSyncPolicy{
			ID: "cancel_p", Name: "Cancel", SourceURL: server.URL,
			TargetGroupID: "p_cancel", Format: "text",
			Config: map[string]string{"allowPrivate": "true"},
		}
		_ = service.CreateSyncPolicy(ctx, policy)

		// 触发同步（由于是 localhost，通过 config 允许）
		_ = service.Sync(ctx, "cancel_p")

		// 等待进入 Running 状态
		for i := 0; i < 50; i++ {
			p, _ := service.GetSyncPolicy(ctx, "cancel_p")
			if p.LastStatus == "running" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		// 执行取消
		service.GetSyncTasks().CancelTask("cancel_p")

		// 验证状态变为已取消
		var pFinal *models.IPSyncPolicy
		for i := 0; i < 50; i++ {
			pFinal, _ = service.GetSyncPolicy(ctx, "cancel_p")
			if pFinal.LastStatus == "cancelled" || pFinal.LastStatus == "failed" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		assert.Equal(t, "cancelled", pFinal.LastStatus)
	})
}
