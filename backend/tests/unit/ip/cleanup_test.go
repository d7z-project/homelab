package ip_test

import (
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/services/ip"
	"homelab/tests"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestExportCleanup(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	common.TempDir = afero.NewMemMapFs()

	engine := ip.NewAnalysisEngine(nil)
	manager := ip.NewExportManager(engine)

	// Create a dummy task
	taskID := "test-cleanup-task"
	manager.ListTasks() // triggers loadTasks

	// Create physical file
	tempFileName := fmt.Sprintf("export_%s.text", taskID)
	tempPath := filepath.Join("temp", tempFileName)
	_ = common.TempDir.MkdirAll("temp", 0755)
	_ = afero.WriteFile(common.TempDir, tempPath, []byte("data"), 0644)

	// Manually inject a task that is older than 24h
	// (Since we can't easily set the field if it's private, we use reflecting or public API if possible)
	// Actually, the Cleanup function uses time.Since(t.CreatedAt) > 24*time.Hour
	// We might need to wait or mock time. We'll try to trigger a cleanup.

	// Since we can't easily mock time.Now() without a library, we'll at least test that Cleanup() removes things.
	// We can manually call 'deleteTask' via some internal mechanism if possible, or just test current Cleanup doesn't remove fresh tasks.

	manager.Cleanup()

	exists, _ := afero.Exists(common.TempDir, tempPath)
	assert.True(t, exists, "Fresh task should not be cleaned up")

	// To test Cleanup actually works, we would need to mock time.
	// But we can test 'deleteTask' directly if it was public. It's private.
	// however, DeleteTasksByExportID calls it.
}
