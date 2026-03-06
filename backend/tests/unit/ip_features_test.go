package unit

import (
	"homelab/tests"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/ip"
	"homelab/pkg/services/actions"
	"homelab/pkg/services/actions/processors"
	"testing"
	"time"
	"net/netip"
	"os"
	"net/http"
	"net/http/httptest"

	"github.com/stretchr/testify/assert"
	"github.com/spf13/afero"
)

func TestIPExportCRUD(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	mmdb := ip.NewMMDBManager()
	service := ip.NewIPPoolService(mmdb)

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
	list, total, err := service.ListExports(ctx, 1, 10, "")
	assert.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Equal(t, "test_export", list[0].ID)

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

	mmdb := ip.NewMMDBManager()
	analysis := ip.NewAnalysisEngine(mmdb)
	manager := ip.NewExportManager(analysis)
	service := ip.NewIPPoolService(mmdb)

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
		if task != nil && task.Status == "Success" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	task := manager.GetTask(taskID)
	assert.NotNil(t, task)
	assert.Equal(t, "Success", task.Status)
	assert.NotEmpty(t, task.ResultURL)

	// Verify text output file exists
	tempPath := "temp/export_" + taskID + ".text"
	exists, _ := afero.Exists(common.TempDir, tempPath)
	assert.True(t, exists)

	// Trigger export (json)
	taskIDJson, _ := manager.TriggerExport(ctx, "export1", "json")
	for i := 0; i < 50; i++ {
		if manager.GetTask(taskIDJson).Status == "Success" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.True(t, manager.GetTask(taskIDJson).Status == "Success")

	// Trigger export (yaml)
	taskIDYaml, _ := manager.TriggerExport(ctx, "export1", "yaml")
	for i := 0; i < 50; i++ {
		if manager.GetTask(taskIDYaml).Status == "Success" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	assert.True(t, manager.GetTask(taskIDYaml).Status == "Success")
}

func TestIPAnalysisEngine(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	mmdb := ip.NewMMDBManager()
	analysis := ip.NewAnalysisEngine(mmdb)
	service := ip.NewIPPoolService(mmdb)

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
	
	// Test graceful handling when MMDB files are missing
	mmdb := ip.NewMMDBManager() // No files loaded
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

	mmdb := ip.NewMMDBManager()
	analysis := ip.NewAnalysisEngine(mmdb)
	manager := ip.NewExportManager(analysis)
	service := ip.NewIPPoolService(mmdb)

	// Create pool
	_ = service.CreateGroup(ctx, &models.IPGroup{ID: "pool1", Name: "Pool 1"})
	common.FS = afero.NewMemMapFs()
	// Write dummy data to prevent fast completion
	codec := ip.NewCodec()
	f, _ := common.FS.Create("network/ip/pools/pool1.bin")
	_ = codec.WritePool(f, []string{"t"}, []ip.Entry{
		{Prefix: netip.MustParsePrefix("1.1.1.1/32"), TagIndices: []uint32{0}},
	})
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

	// 3. Wait a bit for status update
	time.Sleep(150 * time.Millisecond)
	
	t1 := manager.GetTask(taskID1)
	t2 := manager.GetTask(taskID2)
	
	assert.NotNil(t, t1)
	assert.NotNil(t, t2)
}

func TestManagePoolEntry(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	mmdb := ip.NewMMDBManager()
	service := ip.NewIPPoolService(mmdb)

	// Create pool
	group := &models.IPGroup{ID: "pool_entries", Name: "Entry Pool"}
	_ = service.CreateGroup(ctx, group)
	common.FS = afero.NewMemMapFs()

	// 1. Add Entry
	err := service.ManagePoolEntry(ctx, "pool_entries", &models.IPPoolEntryRequest{
		CIDR: "192.168.1.0/24",
		Tags: []string{"tag1"},
	}, "add")
	assert.NoError(t, err)

	// 2. Prevent Duplicate
	err = service.ManagePoolEntry(ctx, "pool_entries", &models.IPPoolEntryRequest{
		CIDR: "192.168.1.0/24",
		Tags: []string{"tag2"},
	}, "add")
	assert.ErrorContains(t, err, "already exists")

	// 3. Update Entry (Tags)
	err = service.ManagePoolEntry(ctx, "pool_entries", &models.IPPoolEntryRequest{
		CIDR: "192.168.1.0/24",
		Tags: []string{"tag1", "tag3"},
	}, "update")
	assert.NoError(t, err)

	// Verify via Preview
	preview, err := service.PreviewPool(ctx, "pool_entries", 0, 10, "")
	assert.NoError(t, err)
	assert.Len(t, preview.Entries, 1)
	assert.ElementsMatch(t, []string{"tag1", "tag3"}, preview.Entries[0].Tags)

	// Verify Search
	searchPreview, err := service.PreviewPool(ctx, "pool_entries", 0, 10, "tag3")
	assert.NoError(t, err)
	assert.Len(t, searchPreview.Entries, 1)

	searchPreviewMiss, err := service.PreviewPool(ctx, "pool_entries", 0, 10, "tag_not_exist")
	assert.NoError(t, err)
	assert.Len(t, searchPreviewMiss.Entries, 0)

	// 4. Delete Entry
	err = service.ManagePoolEntry(ctx, "pool_entries", &models.IPPoolEntryRequest{
		CIDR: "192.168.1.0/24",
	}, "delete")
	assert.NoError(t, err)

	preview2, err := service.PreviewPool(ctx, "pool_entries", 0, 10, "")
	assert.NoError(t, err)
	assert.Len(t, preview2.Entries, 0)
}

func TestIPProcessors(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()

	common.FS = afero.NewMemMapFs()
	common.TempDir = afero.NewMemMapFs()
	actions.Init()

	mmdb := ip.NewMMDBManager()
	service := ip.NewIPPoolService(mmdb)
	processors.RegisterIPProcessors(service, mmdb)

	logger, _ := actions.NewTaskLogger("test-wf", "test-inst")

	// 1. Test IPPoolImportProcessor
	t.Run("IP Pool Import Processor", func(t *testing.T) {
		group := &models.IPGroup{ID: "import_pool", Name: "Import Pool"}
		_ = service.CreateGroup(ctx, group)

		tempFile, _ := os.CreateTemp("", "ip_import_test_*.txt")
		defer os.Remove(tempFile.Name())
		tempFile.WriteString("1.1.1.1/32\n2.2.2.0/24\n")
		tempFile.Close()

		p, ok := actions.GetProcessor("ip/pool/import")
		assert.True(t, ok)

		taskCtx := &actions.TaskContext{
			Context: ctx,
			Logger:  logger,
		}

		_, err := p.Execute(taskCtx, map[string]string{
			"targetPool": "import_pool",
			"filePath":   tempFile.Name(),
			"format":     "text",
			"mode":       "append",
		})
		assert.NoError(t, err)

		// Verify import
		g, _ := service.GetGroup(ctx, "import_pool")
		assert.Equal(t, int64(2), g.EntryCount)
	})

	// 2. Test MMDBDownloadProcessor
	t.Run("MMDB Download Processor", func(t *testing.T) {
		// Mock HTTP Server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("mock mmdb data"))
		}))
		defer server.Close()

		p, ok := actions.GetProcessor("ip/download/mmdb")
		assert.True(t, ok)

		taskCtx := &actions.TaskContext{
			Context: ctx,
			Logger:  logger,
		}

		_, err := p.Execute(taskCtx, map[string]string{
			"url":  server.URL,
			"type": "asn",
		})
		assert.NoError(t, err)

		// Verify file exists in VFS
		exists, _ := afero.Exists(common.FS, ip.MMDBPathASN)
		assert.True(t, exists)
	})
}
