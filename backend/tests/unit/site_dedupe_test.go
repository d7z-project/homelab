package unit

import (
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/site"
	"homelab/tests"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

func TestSiteExportDeduplication(t *testing.T) {
	t.Skip("Deduplication not fully implemented for MVP")
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()
	common.TempDir = afero.NewMemMapFs()

	analysis := site.NewAnalysisEngine(nil)
	manager := site.NewExportManager(analysis)
	service := site.NewSitePoolService(analysis)

	// 1. Setup pool with redundant rules
	group := &models.SiteGroup{Name: "Dedupe Pool"}
	_ = service.CreateGroup(ctx, group)

	// rules:
	// domain:google.com [tag1]
	// full:www.google.com [tag2]
	// domain:mail.google.com [tag3]
	// full:bing.com [tag4]

	_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{Type: 2, Value: "google.com", Tags: []string{"tag1"}}, "add")
	_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{Type: 3, Value: "www.google.com", Tags: []string{"tag2"}}, "add")
	_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{Type: 2, Value: "mail.google.com", Tags: []string{"tag3"}}, "add")
	_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{Type: 3, Value: "bing.com", Tags: []string{"tag4"}}, "add")

	export := &models.SiteExport{
		Name:     "Dedupe Export",
		Rule:     "true",
		GroupIDs: []string{group.ID},
	}
	_ = service.CreateExport(ctx, export)

	// 2. Trigger Export
	taskID, err := manager.TriggerExport(ctx, export.ID, "text")
	assert.NoError(t, err)

	for i := 0; i < 50; i++ {
		task := manager.GetTask(taskID)
		if task != nil && task.Status == "Success" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	// 3. Verify content
	content, _ := afero.ReadFile(common.TempDir, "temp/site_export_"+taskID+".text")
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")

	// Should have exactly 2 lines:
	// domain:google.com (which covers www and mail)
	// full:bing.com
	assert.Len(t, lines, 2)
	assert.Contains(t, lines, "domain:google.com")
	assert.Contains(t, lines, "full:bing.com")

	// 4. Verify Tag Merging (via JSON to easily check tags)
	taskIDJson, _ := manager.TriggerExport(ctx, export.ID, "json")
	for i := 0; i < 50; i++ {
		if manager.GetTask(taskIDJson).Status == "Success" {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	contentJson, _ := afero.ReadFile(common.TempDir, "temp/site_export_"+taskIDJson+".json")
	// The google.com entry should have tag1, tag2, tag3
	assert.Contains(t, string(contentJson), "tag1")
	assert.Contains(t, string(contentJson), "tag2")
	assert.Contains(t, string(contentJson), "tag3")
}
