package unit

import (
	"context"
	"homelab/pkg/models"
	rbacservice "homelab/pkg/services/rbac"
	"homelab/tests"
	"testing"
)

func TestRBACFullWorkflow(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := context.Background()

	// 1. 创建 ServiceAccount
	saName := "test-sa"
	sa, err := rbacservice.CreateServiceAccount(ctx, &models.ServiceAccount{
		Name: saName,
	})
	if err != nil {
		t.Fatalf("CreateServiceAccount failed: %v", err)
	}
	if sa.Token == "" {
		t.Error("Expected token to be generated")
	}

	// 2. 创建 Role
	roleName := "dns-manager"
	_, err = rbacservice.CreateRole(ctx, &models.Role{
		Name: roleName,
		Rules: []models.PolicyRule{
			{
				Resource: "dns/example.com",
				Verbs:    []string{"get", "update"},
			},
		},
	})
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	// 3. 创建 RoleBinding (初始禁用)
	rbName := "test-binding"
	_, err = rbacservice.CreateRoleBinding(ctx, &models.RoleBinding{
		Name:               rbName,
		ServiceAccountName: saName,
		RoleNames:          []string{roleName},
		Enabled:            false,
	})
	if err != nil {
		t.Fatalf("CreateRoleBinding failed: %v", err)
	}

	// 4. 模拟权限 (应为空，因为 Binding 已禁用)
	perms, _ := rbacservice.SimulatePermissions(ctx, saName, "get", "dns")
	if perms.AllowedAll || len(perms.AllowedInstances) > 0 {
		t.Error("Expected no permissions for disabled binding")
	}

	// 5. 启用 Binding 并再次模拟
	_, _ = rbacservice.UpdateRoleBinding(ctx, rbName, &models.RoleBinding{
		Name:               rbName,
		ServiceAccountName: saName,
		RoleNames:          []string{roleName},
		Enabled:            true,
	})

	perms, err = rbacservice.SimulatePermissions(ctx, saName, "get", "dns")
	if err != nil {
		t.Fatalf("SimulatePermissions failed: %v", err)
	}
	
	// 根据 GetPermissions 逻辑，dns/example.com 匹配 resource="dns" 
	// 应将 "example.com" 加入 AllowedInstances
	found := false
	for _, inst := range perms.AllowedInstances {
		if inst == "example.com" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected 'example.com' in AllowedInstances, got %v", perms.AllowedInstances)
	}

	// 6. 重置 Token 验证
	oldToken := sa.Token
	resetSA, err := rbacservice.ResetServiceAccountToken(ctx, saName)
	if err != nil {
		t.Fatalf("Reset token failed: %v", err)
	}
	if resetSA.Token == oldToken {
		t.Error("Token should have changed after reset")
	}

	// 7. 级联删除验证: 删除 Role
	err = rbacservice.DeleteRole(ctx, roleName)
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}
	// RoleBinding 应该被删除 (因为它是唯一的 Role)
	rbResp, _ := rbacservice.ListRoleBindings(ctx, 1, 10, rbName)
	if rbResp.Total > 0 {
		t.Error("RoleBinding should be deleted after its only role is removed")
	}
}
