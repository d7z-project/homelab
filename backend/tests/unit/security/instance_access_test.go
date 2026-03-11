package security_test

import (
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	"homelab/pkg/services/dns"
	"homelab/tests"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstanceLevelRBAC(t *testing.T) {
	ipService, cleanup := tests.SetupIPService()
	defer cleanup()

	// 1. 准备基础数据
	ctxRoot := tests.SetupMockRootContext()

	// 创建两个 IP 池
	poolA := &models.IPPool{ID: "pool-a", Meta: models.IPPoolV1Meta{Name: "Pool A"}}
	poolB := &models.IPPool{ID: "pool-b", Meta: models.IPPoolV1Meta{Name: "Pool B"}}
	_ = ipService.CreateGroup(ctxRoot, poolA)
	_ = ipService.CreateGroup(ctxRoot, poolB)

	// 创建两个域名 (DNS 仍支持包级方法或需同样 Setup)
	domA := &models.Domain{Meta: models.DomainV1Meta{Name: "domain-a.com"}}
	domB := &models.Domain{Meta: models.DomainV1Meta{Name: "domain-b.com"}}
	_ = domA.Bind(nil)
	_ = domB.Bind(nil)
	da, _ := dns.CreateDomain(ctxRoot, domA)
	db, _ := dns.CreateDomain(ctxRoot, domB)

	t.Run("IP: User with specific pool permission", func(t *testing.T) {
		// 用户仅有 pool-a 的更新权限
		userCtx := tests.SetupMockContext("user-a", []models.PolicyRule{
			{Resource: "network/ip/pool-a", Verbs: []string{"update", "get"}},
		})

		// 应该允许更新 pool-a
		poolA.Meta.Name = "Updated Pool A"
		err := ipService.UpdateGroup(userCtx, poolA)
		assert.NoError(t, err)

		// 应该拒绝更新 pool-b (即使没有全局权限)
		poolB.Meta.Name = "Illegal Update"
		err = ipService.UpdateGroup(userCtx, poolB)
		assert.ErrorIs(t, err, commonauth.ErrPermissionDenied)
		assert.Contains(t, err.Error(), "network/ip/pool-b")
	})

	t.Run("DNS: User with specific domain permission", func(t *testing.T) {
		// 用户仅有 domain-a.com 的管理权限
		userCtx := tests.SetupMockContext("user-dns", []models.PolicyRule{
			{Resource: "network/dns/domain-a.com", Verbs: []string{"*"}},
		})

		// 应该允许管理 domain-a
		err := dns.DeleteDomain(userCtx, da.ID)
		assert.NoError(t, err)

		// 应该拒绝管理 domain-b
		err = dns.DeleteDomain(userCtx, db.ID)
		assert.ErrorIs(t, err, commonauth.ErrPermissionDenied)
		assert.Contains(t, err.Error(), "network/dns/domain-b.com")
	})

	t.Run("Global permission override", func(t *testing.T) {
		// 用户拥有全局 network/ip 权限
		globalCtx := tests.SetupMockContext("admin", []models.PolicyRule{
			{Resource: "network/ip", Verbs: []string{"*"}},
		})

		// 应该允许删除 pool-b
		err := ipService.DeleteGroup(globalCtx, "pool-b")
		assert.NoError(t, err)
	})
}
