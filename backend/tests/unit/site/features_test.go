package site_test

import (
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
	"homelab/pkg/services/site"
	"homelab/tests"
	"os"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

type MockLogger struct {
	Logs []string
}

func (l *MockLogger) Log(message string) {
	l.Logs = append(l.Logs, message)
}

func (l *MockLogger) Logf(format string, a ...interface{}) {
	l.Log(fmt.Sprintf(format, a...))
}

func TestSiteMatchers(t *testing.T) {
	// 1. Keyword Matcher
	t.Run("Keyword Matcher", func(t *testing.T) {
		km := site.NewKeywordMatcher()
		km.Insert("google", []string{"g"})
		km.Insert("mail", []string{"m"})

		ok, pat, tags := km.Match("www.google.com")
		assert.True(t, ok)
		assert.Equal(t, "google", pat)
		assert.Contains(t, tags, "g")

		ok, pat, tags = km.Match("mail.google.com")
		assert.True(t, ok)
		assert.ElementsMatch(t, []string{"g", "m"}, tags) // merged tags
	})

	// 2. Regex Matcher
	t.Run("Regex Matcher", func(t *testing.T) {
		rm := site.NewRegexMatcher()
		err := rm.Insert(`^www\.google\..+$`, []string{"google"})
		assert.NoError(t, err)

		ok, pat, tags := rm.Match("www.google.com")
		assert.True(t, ok)
		assert.Equal(t, `^www\.google\..+$`, pat)
		assert.Contains(t, tags, "google")

		ok, _, _ = rm.Match("mail.google.com")
		assert.False(t, ok)
	})
}

func TestSiteAnalysisEngine(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()

	engine := site.NewAnalysisEngine(nil)
	service := site.NewSitePoolService(engine, nil)

	// Create pool and entries
	group := &models.SiteGroup{Name: "Analysis Pool"}
	_ = service.CreateGroup(ctx, group)

	_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{Type: 2, Value: "google.com", NewTags: []string{"search"}}, "add")
	_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{Type: 0, Value: "apple", NewTags: []string{"tech"}}, "add")

	// Hit Test
	t.Run("HitTest Domain", func(t *testing.T) {
		res, err := engine.HitTest(ctx, "mail.google.com", []string{group.ID})
		assert.NoError(t, err)
		assert.True(t, res.Matched)
		assert.Equal(t, uint8(2), res.RuleType)
		assert.Equal(t, "google.com", res.Pattern)
		assert.Contains(t, res.Tags, "search")
	})

	t.Run("HitTest Keyword", func(t *testing.T) {
		res, err := engine.HitTest(ctx, "apple-store.com", []string{group.ID})
		assert.NoError(t, err)
		assert.True(t, res.Matched)
		assert.Equal(t, uint8(0), res.RuleType)
		assert.Contains(t, res.Tags, "tech")
	})
}

func TestSiteExportManager(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()
	common.TempDir = afero.NewMemMapFs()

	analysis := site.NewAnalysisEngine(nil)
	manager := site.NewExportManager(analysis)
	service := site.NewSitePoolService(analysis, manager)

	// Setup data
	group := &models.SiteGroup{Name: "Pool 1"}
	_ = service.CreateGroup(ctx, group)
	_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{Type: 2, Value: "a.com", NewTags: []string{"cn"}}, "add")
	_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{Type: 3, Value: "b.com", NewTags: []string{"us"}}, "add")

	export := &models.SiteExport{
		Name:     "Export 1",
		Rule:     `"cn" in tags`,
		GroupIDs: []string{group.ID},
	}
	_ = service.CreateExport(ctx, export)

	// Trigger
	taskID, err := manager.TriggerExport(ctx, export.ID, "text")
	assert.NoError(t, err)

	// Wait for task
	for i := 0; i < 50; i++ {
		task := manager.GetTask(taskID)
		if task != nil && task.Status == models.TaskStatusSuccess {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}

	task := manager.GetTask(taskID)
	assert.Equal(t, models.TaskStatusSuccess, task.Status)
	assert.NotEmpty(t, task.ResultURL)

	// Verify content
	exists, _ := afero.Exists(common.TempDir, "temp/site_export_"+taskID+".text")
	assert.True(t, exists)
	content, _ := afero.ReadFile(common.TempDir, "temp/site_export_"+taskID+".text")
	assert.Contains(t, string(content), "domain:a.com")
	assert.NotContains(t, string(content), "full:b.com")

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
	manager.WaitAll()
}

func TestSiteImportProcessor(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()
	actions.Init()

	service := site.NewSitePoolService(nil, nil)
	site.RegisterSiteProcessors(service)

	group := &models.SiteGroup{Name: "Import Pool"}
	_ = service.CreateGroup(ctx, group)

	// Create mock file
	tempFile, _ := os.CreateTemp("", "site_imp_*.txt")
	defer os.Remove(tempFile.Name())
	tempFile.WriteString("full:google.com\nkeyword:apple\nregexp:^v2ray\\..+$\nbaidu.com\n")
	tempFile.Close()

	p, ok := actions.GetProcessor("site/pool/import")
	assert.True(t, ok)

	logger, _ := actions.NewTaskLogger("wf", "inst")
	taskCtx := &actions.TaskContext{
		Context: ctx,
		Logger:  logger,
	}

	_, err := p.Execute(taskCtx, map[string]string{
		"targetPool":  group.ID,
		"filePath":    tempFile.Name(),
		"format":      "text",
		"defaultTags": "imported",
	})
	assert.NoError(t, err)

	// Verify
	g, _ := service.GetGroup(ctx, group.ID)
	assert.Equal(t, int64(4), g.EntryCount)

	preview, _ := service.PreviewPool(ctx, group.ID, "", 10, "")
	assert.Len(t, preview.Entries, 4)
}

func TestSitePreviewSearch(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()
	ctx := tests.SetupMockRootContext()
	common.FS = afero.NewMemMapFs()

	service := site.NewSitePoolService(nil, nil)

	group := &models.SiteGroup{ID: "search_pool", Name: "Search Pool"}
	_ = service.CreateGroup(ctx, group)

	_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{Type: 2, Value: "microsoft.com", NewTags: []string{"work"}}, "add")
	_ = service.ManagePoolEntry(ctx, group.ID, &models.SitePoolEntryRequest{Type: 2, Value: "github.com", NewTags: []string{"dev"}}, "add")

	// 1. Search by Value
	res, _ := service.PreviewPool(ctx, group.ID, "", 10, "git")
	assert.Len(t, res.Entries, 1)
	assert.Equal(t, "github.com", res.Entries[0].Value)

	// 2. Search by Tag
	res, _ = service.PreviewPool(ctx, group.ID, "", 10, "work")
	assert.Len(t, res.Entries, 1)
	assert.Equal(t, "microsoft.com", res.Entries[0].Value)

	// 3. Search Miss
	res, _ = service.PreviewPool(ctx, group.ID, "", 10, "apple")
	assert.Len(t, res.Entries, 0)
}
