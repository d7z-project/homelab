package site_test

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/site"
	"homelab/tests"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"bytes"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestSiteSyncPolicy(t *testing.T) {
	service, cleanup := tests.SetupSiteService()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()

	// 1. Create Target Group
	group := &models.SiteGroup{Name: "Target Pool"}
	err := service.CreateGroup(ctx, group)
	assert.NoError(t, err)

	// 2. Setup Mock Server for text sync
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("domain:google.com\nfull:www.baidu.com\nkeyword:test\nregexp:.*\\.org\n"))
	}))
	defer ts.Close()

	// 3. Create Sync Policy
	policy := &models.SiteSyncPolicy{
		Name:          "Test Sync",
		TargetGroupID: group.ID,
		SourceURL:     ts.URL,
		Format:        "text",
		Mode:          "overwrite",
		Cron:          "@every 1h",
		Config:        map[string]string{"tags": "tag1,tag2"},
		Enabled:       true,
	}
	err = service.CreateSyncPolicy(ctx, policy)
	assert.NoError(t, err)
	assert.NotEmpty(t, policy.ID)

	// 4. Trigger Sync
	err = service.Sync(ctx, policy.ID)
	assert.NoError(t, err)

	// Wait for async sync to complete
	time.Sleep(300 * time.Millisecond)

	// 5. Verify Group Data
	res, err := service.PreviewPool(ctx, group.ID, "", 10, "")
	assert.NoError(t, err)
	assert.Len(t, res.Entries, 4)

	// 6. Test Update / Delete
	policy.Name = "Updated Sync"
	err = service.UpdateSyncPolicy(ctx, policy)
	assert.NoError(t, err)

	p2, err := service.GetSyncPolicy(ctx, policy.ID)
	assert.NoError(t, err)
	assert.Equal(t, "Updated Sync", p2.Name)

	// Scan policies
	scanRes, err := service.ScanSyncPolicies(ctx, "", 10, "")
	assert.NoError(t, err)
	assert.Len(t, scanRes.Items, 1)

	// Delete
	err = service.DeleteSyncPolicy(ctx, policy.ID)
	assert.NoError(t, err)
	_, err = service.GetSyncPolicy(ctx, policy.ID)
	assert.Error(t, err)
}

func TestSiteSyncPolicy_GeoSite(t *testing.T) {
	service, cleanup := tests.SetupSiteService()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()

	group := &models.SiteGroup{Name: "Target Pool 2"}
	err := service.CreateGroup(ctx, group)
	assert.NoError(t, err)

	entry := site.ParsedGeoSiteEntry{
		Category: "cn",
		Type:     2,
		Value:    "qq.com",
	}
	var buf bytes.Buffer
	_ = site.BuildV2RayGeoSite(&buf, map[string][]site.ParsedGeoSiteEntry{
		"cn": {entry},
	})
	data := buf.Bytes()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write(data)
	}))
	defer ts.Close()

	policy := &models.SiteSyncPolicy{
		Name:          "Test GeoSite Sync",
		TargetGroupID: group.ID,
		SourceURL:     ts.URL,
		Format:        "geosite",
		Mode:          "append",
		Config:        map[string]string{"category": "cn"},
	}
	err = service.CreateSyncPolicy(ctx, policy)
	assert.NoError(t, err)

	err = service.Sync(ctx, policy.ID)
	assert.NoError(t, err)
	
	time.Sleep(300 * time.Millisecond)

	res, err := service.PreviewPool(ctx, group.ID, "", 10, "")
	assert.NoError(t, err)
	assert.Len(t, res.Entries, 1)
	assert.Equal(t, "qq.com", res.Entries[0].Value)
	assert.Contains(t, res.Entries[0].Tags, "cn")
}
