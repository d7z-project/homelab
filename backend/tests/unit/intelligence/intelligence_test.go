package intelligence_test

import (
	"fmt"
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

func TestIntelligenceService_Base(t *testing.T) {
	service, cleanup := tests.SetupIntelligenceService()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	t.Run("Source CRUD and Simple Sync", func(t *testing.T) {
		common.FS = afero.NewMemMapFs()
		source := &models.IntelligenceSource{Meta: models.IntelligenceSourceV1Meta{
			Name: "Test Source",
			Type: "asn",
			URL:  "http://example.com/asn.mmdb", Config: map[string]string{"allowPrivate": "true"}},
		}
		err := service.CreateSource(ctx, source)
		assert.NoError(t, err)
		assert.NotEmpty(t, source.ID)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(make([]byte, 100))
		}))
		defer server.Close()

		source.Meta.URL = server.URL
		_ = service.UpdateSource(ctx, source)

		err = service.SyncSource(ctx, source.ID)
		assert.NoError(t, err)

		var s *models.IntelligenceSource
		for i := 0; i < 50; i++ {
			s, _ = repo.SourceRepo.Get(ctx, source.ID)
			if s.Status.Status == models.TaskStatusSuccess || s.Status.Status == models.TaskStatusFailed {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		assert.Equal(t, models.TaskStatusSuccess, s.Status.Status)
		exists, _ := afero.Exists(common.FS, fmt.Sprintf("%s/%s.mmdb", ip.MMDBDir, s.ID))
		assert.True(t, exists)
	})
}

func TestIntelligence_FrameworkIntegration(t *testing.T) {
	service, cleanup := tests.SetupIntelligenceService()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	t.Run("Reconcile Zombie Task", func(t *testing.T) {
		source := &models.IntelligenceSource{Meta: models.IntelligenceSourceV1Meta{Name: "Zombie"}}
		_ = service.CreateSource(ctx, source)
		task := &intelligence.SyncTask{ID: source.ID, Status: models.TaskStatusRunning, CreatedAt: time.Now()}
		service.GetTasks().AddTask(task)
		service.GetTasks().Reconcile(ctx)
		retrieved, _ := service.GetTasks().GetTask(source.ID)
		assert.Equal(t, models.TaskStatusFailed, retrieved.GetStatus())
	})

	t.Run("Cancellation of Sync Task", func(t *testing.T) {
		var downloadStarted = make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(downloadStarted)
			for {
				select {
				case <-r.Context().Done():
					return
				default:
					w.Write([]byte("chunk\n"))
					time.Sleep(10 * time.Millisecond)
				}
			}
		}))
		defer server.Close()

		source := &models.IntelligenceSource{ID: "cancel_src", Meta: models.IntelligenceSourceV1Meta{
			Name: "Cancel", URL: server.URL, Type: "city", Config: map[string]string{"allowPrivate": "true"}},
		}
		_ = service.CreateSource(ctx, source)
		_ = service.SyncSource(ctx, source.ID)

		select {
		case <-downloadStarted:
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout")
		}

		service.GetTasks().CancelTask(source.ID)

		var sFinal *models.IntelligenceSource
		for i := 0; i < 50; i++ {
			sFinal, _ = repo.SourceRepo.Get(ctx, source.ID)
			if sFinal.Status.Status == models.TaskStatusCancelled {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		assert.Equal(t, models.TaskStatusCancelled, sFinal.Status.Status)
	})
}