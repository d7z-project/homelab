package ip_test

import (
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

func TestIPSyncFormats(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	common.FS = afero.NewMemMapFs()
	mmdb := ip.NewMMDBManager()
	service := ip.NewIPPoolService(mmdb)
	ctx := commonauth.WithPermissions(tests.SetupMockRootContext(), &models.ResourcePermissions{AllowedAll: true})

	group := &models.IPGroup{ID: "format_pool", Name: "Format Pool"}
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

	t.Run("Sync CSV Format", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "ip,tag")
			fmt.Fprintln(w, "1.2.3.4,test-tag")
			fmt.Fprintln(w, "5.6.7.8/24,another-tag")
		}))
		defer server.Close()

		policy := &models.IPSyncPolicy{
			ID: "csv_policy", Name: "CSV", SourceURL: server.URL, Format: "csv", TargetGroupID: "format_pool",
			Config: map[string]string{
				"allowPrivate": "true",
				"ipColumn":     "0",
				"tagColumn":    "1",
				"tags":         "global1,global2",
			},
		}
		_ = service.CreateSyncPolicy(ctx, policy)
		syncAndWait("csv_policy")

		res, _ := service.PreviewPool(ctx, "format_pool", 0, 10, "")
		found := false
		for _, e := range res.Entries {
			if e.CIDR == "1.2.3.4/32" {
				assert.Contains(t, e.Tags, "test-tag")
				assert.Contains(t, e.Tags, "global1")
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("Sync GeoIP-DAT Format", func(t *testing.T) {
		var buf bytes.Buffer
		groups := map[string][]netip.Prefix{
			"CN": {netip.MustParsePrefix("114.114.114.114/32")},
		}
		_ = ip.BuildV2RayGeoIP(&buf, groups)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(buf.Bytes())
		}))
		defer server.Close()

		policy := &models.IPSyncPolicy{
			ID: "v2ray_policy", Name: "V2Ray", SourceURL: server.URL, Format: "geoip-dat", TargetGroupID: "format_pool",
			Config: map[string]string{
				"allowPrivate": "true",
				"code":         "CN",
			},
		}
		_ = service.CreateSyncPolicy(ctx, policy)
		syncAndWait("v2ray_policy")

		res, _ := service.PreviewPool(ctx, "format_pool", 0, 100, "")
		found := false
		for _, e := range res.Entries {
			if e.CIDR == "114.114.114.114/32" {
				assert.Contains(t, e.Tags, "cn")
				found = true
			}
		}
		assert.True(t, found)
	})
}
