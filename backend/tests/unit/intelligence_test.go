package unit

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/intelligence"
	"homelab/pkg/services/intelligence"
	"homelab/pkg/services/ip"
	"homelab/tests"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIntelligenceService(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()

	mmdb := ip.NewMMDBManager()
	service := intelligence.NewIntelligenceService(mmdb)

	// 1. Create Source
	source := &models.IntelligenceSource{
		Name: "Test Source",
		Type: "asn",
		URL:  "http://example.com/asn.mmdb",
	}
	err := service.CreateSource(ctx, source)
	assert.NoError(t, err)
	assert.NotEmpty(t, source.ID)

	// 2. List Sources
	sources, err := service.ListSources(ctx)
	assert.NoError(t, err)
	assert.Len(t, sources, 1)
	assert.Equal(t, "Test Source", sources[0].Name)

	// 3. Sync Source (Mock Download)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("mock mmdb content"))
	}))
	defer server.Close()

	source.URL = server.URL
	_ = service.UpdateSource(ctx, source)

	err = service.SyncSource(ctx, source.ID)
	assert.NoError(t, err)

	// Wait for async download
	for i := 0; i < 10; i++ {
		s, _ := repo.GetSource(ctx, source.ID)
		if s.Status == "Ready" {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	s, _ := repo.GetSource(ctx, source.ID)
	assert.Equal(t, "Ready", s.Status)
	assert.False(t, s.LastUpdatedAt.IsZero())

	// Verify file in VFS
	exists, _ := afero.Exists(common.FS, ip.MMDBPathASN)
	assert.True(t, exists)

	// 4. Delete Source
	err = service.DeleteSource(ctx, source.ID)
	assert.NoError(t, err)
	sources, _ = service.ListSources(ctx)
	assert.Len(t, sources, 0)
}
