package tests

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsrepo "homelab/pkg/repositories/dns"
	rbacrepo "homelab/pkg/repositories/rbac"
	"homelab/pkg/services/actions"
	_ "homelab/pkg/services/actions/processors"
	dnsservice "homelab/pkg/services/dns"
	"homelab/pkg/services/intelligence"
	"homelab/pkg/services/ip"
	"homelab/pkg/services/site"
	"log"

	"gopkg.d7z.net/middleware/kv"
	"gopkg.d7z.net/middleware/lock"
)

// SetupTestDB 初始化一个内存数据库用于测试
// 返回一个清理函数，用于在测试结束时关闭数据库
func SetupTestDB() func() {
	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		log.Fatalf("failed to create test db: %v", err)
	}

	// 保存旧的引用以便恢复
	oldDB := common.DB
	oldLocker := common.Locker
	oldFS := common.FS
	oldTemp := common.TempDir
	common.DB = db

	// Clear caches (must be after common.DB is set because they write to DB)
	dnsrepo.ClearCache()
	rbacrepo.ClearCache()
	dnsservice.ClearCache()

	locker, _ := lock.NewLocker("memory://")
	common.Locker = locker

	// Use InitVFS to get sandboxed memory FS for tests
	fs, _ := common.InitVFS("memory://")
	common.FS = fs

	temp, _ := common.InitVFS("memory://")
	common.TempDir = temp

	actions.Init()

	return func() {
		db.Close()
		common.DB = oldDB
		common.Locker = oldLocker
		common.FS = oldFS
		common.TempDir = oldTemp
	}
}

// SetupMockRootContext 返回一个具有 Root (AllowedAll) 权限的上下文
func SetupMockRootContext() context.Context {
	return auth.WithPermissions(context.Background(), &models.ResourcePermissions{
		AllowedAll: true,
	})
}

// SetupMockContext 返回一个具有特定实例权限的上下文
func SetupMockContext(userID string, rules []models.PolicyRule) context.Context {
	allowedInstances := []string{}
	for _, r := range rules {
		allowedInstances = append(allowedInstances, r.Resource)
	}
	return auth.WithPermissions(context.Background(), &models.ResourcePermissions{
		AllowedAll:       false,
		AllowedInstances: allowedInstances,
	})
}

// SetupIPService 初始化 IPPoolService 及其依赖
func SetupIPService() (*ip.IPPoolService, func()) {
	cleanup := SetupTestDB()
	mmdb := ip.NewMMDBManager()
	service := ip.NewIPPoolService(mmdb)
	return service, cleanup
}

// SetupSiteService 初始化 SitePoolService 及其依赖
func SetupSiteService() (*site.SitePoolService, func()) {
	cleanup := SetupTestDB()
	mmdb := ip.NewMMDBManager()
	engine := site.NewAnalysisEngine(mmdb)
	service := site.NewSitePoolService(engine)
	return service, cleanup
}

// SetupIntelligenceService 初始化 IntelligenceService 及其依赖
func SetupIntelligenceService() (*intelligence.IntelligenceService, func()) {
	cleanup := SetupTestDB()
	mmdb := ip.NewMMDBManager()
	service := intelligence.NewIntelligenceService(mmdb)
	return service, cleanup
}

// MockProcessor 用于测试的模拟处理器
type MockProcessor struct {
	ExecuteFunc func(ctx *actions.TaskContext, inputs map[string]string) (map[string]string, error)
	MockID      string
}

func (m *MockProcessor) Manifest() actions.StepManifest {
	id := m.MockID
	if id == "" {
		id = "test/mock"
	}
	return actions.StepManifest{
		ID:          id,
		Name:        "Mock Processor",
		Description: "A processor for testing purposes.",
		Params: []models.ParamDefinition{
			{Name: "input", Optional: true},
			{Name: "input_val", Optional: true},
		},
		OutputParams: []models.ParamDefinition{
			{Name: "output"},
			{Name: "out_val"},
		},
	}
}

func (m *MockProcessor) Execute(ctx *actions.TaskContext, inputs map[string]string) (map[string]string, error) {
	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, inputs)
	}
	res := make(map[string]string)
	if val, ok := inputs["input"]; ok {
		res["output"] = val
	}
	if val, ok := inputs["input_val"]; ok {
		res["out_val"] = val + "_processed"
	}
	return res, nil
}
