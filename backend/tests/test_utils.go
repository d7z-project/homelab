package tests

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsrepo "homelab/pkg/repositories/dns"
	rbacrepo "homelab/pkg/repositories/rbac"
	"homelab/pkg/services/actions"
	dnsservice "homelab/pkg/services/dns"
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
