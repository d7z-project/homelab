package intelligence_test

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

func TestIntelligenceService_Base(t *testing.T) {
	service, cleanup := tests.SetupIntelligenceService()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	t.Run("Source CRUD and Simple Sync", func(t *testing.T) {
		common.FS = afero.NewMemMapFs()
		// 1. Create Source
		source := &models.IntelligenceSource{
			Name: "Test Source",
			Type: "asn",
			URL:  "http://example.com/asn.mmdb", Config: map[string]string{"allowPrivate": "true"},
		}
		err := service.CreateSource(ctx, source)
		assert.NoError(t, err)
		assert.NotEmpty(t, source.ID)

		// 2. Sync Source (Mock Download)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write(make([]byte, 100)) // 模拟 mmdb 数据
		}))
		defer server.Close()

		source.URL = server.URL
		_ = service.UpdateSource(ctx, source)

		err = service.SyncSource(ctx, source.ID)
		assert.NoError(t, err)

		// Wait for completion (Ready or Error)
		var s *models.IntelligenceSource
		for i := 0; i < 50; i++ {
			s, _ = repo.GetSource(ctx, source.ID)
			if s.Status == "Ready" || s.Status == "Error" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		assert.Equal(t, "Ready", s.Status, "Sync failed: "+s.ErrorMessage)

		// Verify file in VFS
		exists, _ := afero.Exists(common.FS, ip.MMDBPathASN)
		assert.True(t, exists, "ASN file should be created in VFS")
	})

	t.Run("SSRF Protection", func(t *testing.T) {
		source := &models.IntelligenceSource{
			Name: "SSRF Source",
			Type: "asn",
			URL:  "http://localhost/secret",
		}
		_ = service.CreateSource(ctx, source)

		_ = service.SyncSource(ctx, source.ID)

		var s *models.IntelligenceSource
		for i := 0; i < 50; i++ {
			s, _ = repo.GetSource(ctx, source.ID)
			if s.Status == "Error" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		assert.Equal(t, "Error", s.Status)
		assert.Contains(t, s.ErrorMessage, "SSRF detected")
	})
}

func TestIntelligence_FrameworkIntegration(t *testing.T) {
	service, cleanup := tests.SetupIntelligenceService()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	t.Run("Reconcile Zombie Task", func(t *testing.T) {
		source := &models.IntelligenceSource{ID: "zombie_src", Name: "Zombie"}
		_ = service.CreateSource(ctx, source)

		task := &intelligence.SyncTask{ID: "zombie_src", Status: "Running", CreatedAt: time.Now()}
		service.GetTasks().AddTask(task)

		service.GetTasks().Reconcile(ctx)

		retrieved, _ := service.GetTasks().GetTask("zombie_src")
		assert.Equal(t, "Failed", retrieved.GetStatus())
	})

	t.Run("Cancellation of Sync Task", func(t *testing.T) {
		var downloadStarted = make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			close(downloadStarted)
			// 无限循环下载
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

		source := &models.IntelligenceSource{
			ID: "cancel_src", Name: "Cancel", URL: server.URL, Type: "city", Config: map[string]string{"allowPrivate": "true"},
		}
		_ = service.CreateSource(ctx, source)

		_ = service.SyncSource(ctx, source.ID)

		// 等待下载实际开始
		select {
		case <-downloadStarted:
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for download to start")
		}

		// 确保状态已经切到 Running
		for i := 0; i < 50; i++ {
			t, _ := service.GetTasks().GetTask(source.ID)
			if t.GetStatus() == "Running" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		// 执行取消
		service.GetTasks().CancelTask(source.ID)

		// 验证 Task 状态
		var tFinal models.TaskInfo
		for i := 0; i < 50; i++ {
			tf, _ := service.GetTasks().GetTask(source.ID)
			if tf.GetStatus() == "Cancelled" {
				tFinal = tf
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		assert.NotNil(t, tFinal)
		assert.Equal(t, "Cancelled", tFinal.GetStatus())

		// 验证 Source 状态
		var sFinal *models.IntelligenceSource
		for i := 0; i < 50; i++ {
			sFinal, _ = repo.GetSource(ctx, source.ID)
			if sFinal.Status == "Cancelled" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
		assert.Equal(t, "Cancelled", sFinal.Status)
	})
}
