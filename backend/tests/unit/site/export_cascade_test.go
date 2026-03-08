package site_test

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/site"
	"homelab/tests"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestSiteExportCascadeDeleteAndCleanup(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()
	common.TempDir = afero.NewMemMapFs()

	engine := site.NewAnalysisEngine(nil)
	manager := site.NewExportManager(engine)
	service := site.NewSitePoolService(engine, manager)

	// Create Group
	group := &models.SiteGroup{Name: "Cascade Pool"}
	err := service.CreateGroup(ctx, group)
	assert.NoError(t, err)

	// Create Export
	export := &models.SiteExport{Name: "Cascade Export", Rule: "true", GroupIDs: []string{group.ID}}
	err = service.CreateExport(ctx, export)
	assert.NoError(t, err)

	// Trigger Export
	taskID, err := manager.TriggerExport(ctx, export.ID, "text")
	assert.NoError(t, err)

	// Wait for the task to finish
	for i := 0; i < 50; i++ {
		t := manager.GetTask(taskID)
		if t != nil && (t.Status == "Success" || t.Status == "Failed" || t.Status == "Cancelled") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	task := manager.GetTask(taskID)
	assert.NotNil(t, task)

	// Verify tasks exist
	tasks := manager.ListTasks()
	assert.Len(t, tasks, 1)

	// Trigger again to test cascade
	taskID2, _ := manager.TriggerExport(ctx, export.ID, "text")
	for i := 0; i < 50; i++ {
		t := manager.GetTask(taskID2)
		if t != nil && (t.Status == "Success" || t.Status == "Failed" || t.Status == "Cancelled") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	tasks = manager.ListTasks()
	assert.Len(t, tasks, 2)

	// Delete Export - should trigger DeleteTasksByExportID
	err = service.DeleteExport(ctx, export.ID)
	assert.NoError(t, err)

	tasks = manager.ListTasks()
	assert.Len(t, tasks, 0) // Should be cascadingly deleted

	// Allow background saveTasks to complete before teardown
	manager.WaitAll()
}
