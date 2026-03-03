package tests

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsrepo "homelab/pkg/repositories/dns"
	rbacrepo "homelab/pkg/repositories/rbac"
	dnsservice "homelab/pkg/services/dns"
	"log"

	"gopkg.d7z.net/middleware/kv"
)

// SetupTestDB 初始化一个内存数据库用于测试
// 返回一个清理函数，用于在测试结束时关闭数据库
func SetupTestDB() func() {
	// Clear caches
	dnsrepo.ClearCache()
	rbacrepo.ClearCache()
	dnsservice.ClearCache()

	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		log.Fatalf("failed to create test db: %v", err)
	}

	// 保存旧的 DB 引用以便恢复（如果需要）
	oldDB := common.DB
	common.DB = db

	return func() {
		db.Close()
		// No need to restore oldDB here if it might be nil,
		// but let's be safe.
		common.DB = oldDB
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
