package ip_test

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/ip"
	"homelab/tests"
	"net/netip"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestIPExportCRUD(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	mmdb := ip.NewMMDBManager(nil)
	analysis := ip.NewAnalysisEngine(mmdb)
	manager := ip.NewExportManager(analysis)
	service := ip.NewIPPoolService(analysis, manager)
	ctx := tests.SetupMockRootContext()

	export := &models.IPExport{
		ID:       "test_export",
		Name:     "Test Export",
		Rule:     "true",
		GroupIDs: []string{"pool1"},
	}

	// Create
	err := service.CreateExport(ctx, export)
	assert.NoError(t, err)

	// Get
	e, err := service.GetExport(ctx, "test_export")
	assert.NoError(t, err)
	assert.Equal(t, "Test Export", e.Name)

	// List
	res, err := service.ScanExports(ctx, "", 10, "")
	assert.NoError(t, err)
	assert.Len(t, res.Items, 1)
	assert.Equal(t, "test_export", res.Items[0].ID)

	// Update
	e.Name = "Updated Export"
	err = service.UpdateExport(ctx, e)
	assert.NoError(t, err)

	e2, _ := service.GetExport(ctx, "test_export")
	assert.Equal(t, "Updated Export", e2.Name)

	// Delete
	err = service.DeleteExport(ctx, "test_export")
	assert.NoError(t, err)

	_, err = service.GetExport(ctx, "test_export")
	assert.Error(t, err)
}

func TestIPExportManager(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	// Initialize global VFS for export tasks
	common.TempDir = afero.NewMemMapFs()

	mmdb := ip.NewMMDBManager(nil)
	analysis := ip.NewAnalysisEngine(mmdb)
	manager := ip.NewExportManager(analysis)
	service := ip.NewIPPoolService(analysis, manager)

	// 1. Create a group and add some data
	group := &models.IPGroup{ID: "pool1", Name: "Pool 1"}
	_ = service.CreateGroup(ctx, group)

	// Write mock data to VFS
	common.FS = afero.NewMemMapFs()
	codec := ip.NewCodec()
	f, _ := common.FS.Create("network/ip/pools/pool1.bin")
	_ = codec.WritePool(f, []string{"cn", "us"}, []ip.Entry{
		{Prefix: netip.MustParsePrefix("8.8.8.8/32"), TagIndices: []uint32{1}},
		{Prefix: netip.MustParsePrefix("114.114.114.114/32"), TagIndices: []uint32{0}},
	})
	f.Close()

	// 2. Create export configuration
	export := &models.IPExport{
		ID:       "export1",
		Name:     "Export 1",
		Rule:     `"cn" in tags`,
		GroupIDs: []string{"pool1"},
	}
	_ = service.CreateExport(ctx, export)

	// 3. Trigger export (text)
	taskID, err := manager.TriggerExport(ctx, "export1", "text")
	assert.NoError(t, err)
	assert.NotEmpty(t, taskID)

	// Wait for task to complete properly
	for i := 0; i < 50; i++ {
		task := manager.GetTask(taskID)
		if task != nil && task.Status == models.TaskStatusSuccess {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	task := manager.GetTask(taskID)
	assert.NotNil(t, task)
	assert.Equal(t, models.TaskStatusSuccess, task.Status)
	assert.NotEmpty(t, task.ResultURL)

	// Verify text output file exists
	tempPath := "temp/export_" + taskID + ".text"
	exists, _ := afero.Exists(common.TempDir, tempPath)
	assert.True(t, exists)

	// Trigger export (json)
	taskIDJson, _ := manager.TriggerExport(ctx, "export1", "json")
	for i := 0; i < 50; i++ {
		if manager.GetTask(taskIDJson).Status == models.TaskStatusSuccess {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.True(t, manager.GetTask(taskIDJson).Status == models.TaskStatusSuccess)

	// Trigger export (yaml)
	taskIDYaml, _ := manager.TriggerExport(ctx, "export1", "yaml")
	for i := 0; i < 50; i++ {
		if manager.GetTask(taskIDYaml).Status == models.TaskStatusSuccess {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.True(t, manager.GetTask(taskIDYaml).Status == models.TaskStatusSuccess)

	// Allow background saveTasks to complete before teardown
	for i := 0; i < 50; i++ {
		allDone := true
		for _, task := range manager.ListTasks() {
			if task.Status == models.TaskStatusRunning || task.Status == models.TaskStatusPending {
				allDone = false
				break
			}
		}
		if allDone {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	// Extra buffer to let saveTasks() finish after Status updates
	manager.WaitAll()
}

func TestIPAnalysisEngine(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	mmdb := ip.NewMMDBManager(nil)
	analysis := ip.NewAnalysisEngine(mmdb)
	manager := ip.NewExportManager(analysis)
	service := ip.NewIPPoolService(analysis, manager)

	// 1. Create group and mock data
	group := &models.IPGroup{ID: "pool_analysis", Name: "Analysis Pool"}
	_ = service.CreateGroup(ctx, group)

	common.FS = afero.NewMemMapFs()
	common.FS.MkdirAll("network/ip/pools", 0755)
	codec := ip.NewCodec()
	f, _ := common.FS.Create("network/ip/pools/pool_analysis.bin")
	_ = codec.WritePool(f, []string{"malicious"}, []ip.Entry{
		{Prefix: netip.MustParsePrefix("1.2.3.4/32"), TagIndices: []uint32{0}},
	})
	f.Close()

	// 2. Test HitTest
	res, err := analysis.HitTest(ctx, "1.2.3.4", []string{"pool_analysis"})
	assert.NoError(t, err)
	assert.True(t, res.Matched)
	assert.Equal(t, "1.2.3.4/32", res.CIDR)
	assert.Contains(t, res.Tags, "malicious")

	res2, err := analysis.HitTest(ctx, "8.8.8.8", []string{"pool_analysis"})
	assert.NoError(t, err)
	assert.False(t, res2.Matched)
}

func TestIPInfoLookup(t *testing.T) {
	// Setup FS to prevent nil pointer in afero.ReadFile
	common.FS = afero.NewMemMapFs()

	mmdb := ip.NewMMDBManager(nil)
	analysis := ip.NewAnalysisEngine(mmdb)
	ctx := tests.SetupMockRootContext()

	info, err := analysis.Info(ctx, "8.8.8.8")
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, "8.8.8.8", info.IP)
	// Other fields should be empty/zero
	assert.Empty(t, info.Country)
	assert.Empty(t, info.City)
	assert.Zero(t, info.ASN)
}

func TestIPExportCancellation(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	// IMPORTANT: Initialize TempDir for export tasks
	common.TempDir = afero.NewMemMapFs()

	mmdb := ip.NewMMDBManager(nil)
	analysis := ip.NewAnalysisEngine(mmdb)
	manager := ip.NewExportManager(analysis)
	service := ip.NewIPPoolService(analysis, manager)

	// Create pool
	_ = service.CreateGroup(ctx, &models.IPGroup{ID: "pool1", Name: "Pool 1"})
	common.FS = afero.NewMemMapFs()
	// Write dummy data to prevent fast completion
	codec := ip.NewCodec()
	f, _ := common.FS.Create("network/ip/pools/pool1.bin")
	var dummyEntries []ip.Entry
	for i := 0; i < 100000; i++ {
		dummyEntries = append(dummyEntries, ip.Entry{Prefix: netip.MustParsePrefix("1.1.1.1/32"), TagIndices: []uint32{0}})
	}
	_ = codec.WritePool(f, []string{"t"}, dummyEntries)
	f.Close()

	export := &models.IPExport{
		ID:       "long_export",
		Name:     "Long Export",
		Rule:     "true",
		GroupIDs: []string{"pool1"},
	}
	_ = service.CreateExport(ctx, export)

	// 1. Trigger First
	taskID1, err := manager.TriggerExport(ctx, "long_export", "text")
	assert.NoError(t, err)

	// 2. Trigger Second immediately
	taskID2, err := manager.TriggerExport(ctx, "long_export", "text")
	assert.NoError(t, err)
	assert.NotEqual(t, taskID1, taskID2)

	// 3. Wait for second task to complete
	for i := 0; i < 50; i++ {
		t := manager.GetTask(taskID2)
		if t != nil && (t.Status == models.TaskStatusSuccess || t.Status == models.TaskStatusFailed || t.Status == models.TaskStatusCancelled) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	t1 := manager.GetTask(taskID1)
	t2 := manager.GetTask(taskID2)

	assert.NotNil(t, t1)
	assert.NotNil(t, t2)
	assert.Equal(t, models.TaskStatusCancelled, t1.Status)

	// Allow background saveTasks to finish
	manager.WaitAll()
}

func TestIPExportManualCancellation(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.TempDir = afero.NewMemMapFs()

	mmdb := ip.NewMMDBManager(nil)
	analysis := ip.NewAnalysisEngine(mmdb)
	manager := ip.NewExportManager(analysis)
	service := ip.NewIPPoolService(analysis, manager)

	_ = service.CreateGroup(ctx, &models.IPGroup{ID: "pool1", Name: "Pool 1"})
	common.FS = afero.NewMemMapFs()
	codec := ip.NewCodec()
	f, _ := common.FS.Create("network/ip/pools/pool1.bin")
	var dummyEntries []ip.Entry
	for i := 0; i < 20000; i++ {
		dummyEntries = append(dummyEntries, ip.Entry{Prefix: netip.MustParsePrefix("1.1.1.1/32"), TagIndices: []uint32{0}})
	}
	_ = codec.WritePool(f, []string{"t"}, dummyEntries)
	f.Close()

	export := &models.IPExport{ID: "long_export", Name: "Long Export", Rule: "true", GroupIDs: []string{"pool1"}}
	_ = service.CreateExport(ctx, export)

	taskID, _ := manager.TriggerExport(ctx, "long_export", "text")

	// Wait for Running
	for i := 0; i < 50; i++ {
		t := manager.GetTask(taskID)
		if t != nil && t.Status == models.TaskStatusRunning {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Manual Cancel
	ok := manager.CancelTask(taskID)
	assert.True(t, ok)

	// Wait for Cancelled
	for i := 0; i < 50; i++ {
		t := manager.GetTask(taskID)
		if t != nil && (t.Status == models.TaskStatusCancelled || t.Status == models.TaskStatusFailed) {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.Equal(t, models.TaskStatusCancelled, manager.GetTask(taskID).Status)
	manager.WaitAll()
}

func TestManagePoolEntry(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	mmdb := ip.NewMMDBManager(nil)
	analysis := ip.NewAnalysisEngine(mmdb)
	service := ip.NewIPPoolService(analysis, nil)

	// Create pool
	group := &models.IPGroup{ID: "pool_entries", Name: "Entry Pool"}
	_ = service.CreateGroup(ctx, group)
	common.FS = afero.NewMemMapFs()

	// 1. Add Tag
	err := service.ManagePoolEntry(ctx, "pool_entries", &models.IPPoolEntryRequest{
		CIDR:    "192.168.1.0/24",
		NewTags: []string{"tag1"},
	}, "add")
	assert.NoError(t, err)

	// 2. Add another tag to same CIDR
	err = service.ManagePoolEntry(ctx, "pool_entries", &models.IPPoolEntryRequest{
		CIDR:    "192.168.1.0/24",
		NewTags: []string{"tag2"},
	}, "add")
	assert.NoError(t, err)

	// 3. Update Tag (Rename tag1 to tag3)
	err = service.ManagePoolEntry(ctx, "pool_entries", &models.IPPoolEntryRequest{
		CIDR:    "192.168.1.0/24",
		OldTags: []string{"tag1"},
		NewTags: []string{"tag3"},
	}, "update")
	assert.NoError(t, err)

	// Verify via Preview
	preview, err := service.PreviewPool(ctx, "pool_entries", 0, 10, "")
	assert.NoError(t, err)
	assert.Len(t, preview.Entries, 1)
	assert.ElementsMatch(t, []string{"tag2", "tag3"}, preview.Entries[0].Tags)

	// Verify Search
	searchPreview, err := service.PreviewPool(ctx, "pool_entries", 0, 10, "tag3")
	assert.NoError(t, err)
	assert.Len(t, searchPreview.Entries, 1)

	searchPreviewMiss, err := service.PreviewPool(ctx, "pool_entries", 0, 10, "tag_not_exist")
	assert.NoError(t, err)
	assert.Len(t, searchPreviewMiss.Entries, 0)

	// 4. Delete Tag
	err = service.ManagePoolEntry(ctx, "pool_entries", &models.IPPoolEntryRequest{
		CIDR:    "192.168.1.0/24",
		OldTags: []string{"tag2"},
	}, "delete")
	assert.NoError(t, err)

	preview2, err := service.PreviewPool(ctx, "pool_entries", 0, 10, "")
	assert.NoError(t, err)
	assert.Len(t, preview2.Entries, 1)
	assert.ElementsMatch(t, []string{"tag3"}, preview2.Entries[0].Tags)

	// 5. Delete Entry Entirely
	err = service.ManagePoolEntry(ctx, "pool_entries", &models.IPPoolEntryRequest{
		CIDR: "192.168.1.0/24",
	}, "delete")
	assert.NoError(t, err)

	preview3, err := service.PreviewPool(ctx, "pool_entries", 0, 10, "")
	assert.NoError(t, err)
	assert.Len(t, preview3.Entries, 0)
}
