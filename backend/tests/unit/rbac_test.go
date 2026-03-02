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

	// 1. 创建 ServiceAccount (使用显式 ID)
	saID := "test-sa-01"
	sa, err := rbacservice.CreateServiceAccount(ctx, &models.ServiceAccount{
		ID:   saID,
		Name: "Test SA",
	})
	if err != nil {
		t.Fatalf("CreateServiceAccount failed: %v", err)
	}
	if sa.Token == "" {
		t.Error("Expected token to be generated")
	}
	if sa.ID != saID {
		t.Errorf("Expected ID %s, got %s", saID, sa.ID)
	}

	// 2. 创建 Role (UUID 自动生成)
	role, err := rbacservice.CreateRole(ctx, &models.Role{
		Name: "DNS Manager",
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
	if role.ID == "" {
		t.Error("Expected Role UUID to be generated")
	}
	roleID := role.ID

	// 3. 创建 RoleBinding (初始禁用)
	rb, err := rbacservice.CreateRoleBinding(ctx, &models.RoleBinding{
		Name:             "Test Binding",
		ServiceAccountID: saID,
		RoleIDs:          []string{roleID},
		Enabled:          false,
	})
	if err != nil {
		t.Fatalf("CreateRoleBinding failed: %v", err)
	}
	if rb.ID == "" {
		t.Error("Expected RoleBinding UUID to be generated")
	}
	rbID := rb.ID

	// 4. 模拟权限 (应为空，因为 Binding 已禁用)
	perms, _ := rbacservice.SimulatePermissions(ctx, saID, "get", "dns")
	if perms.AllowedAll || len(perms.AllowedInstances) > 0 {
		t.Error("Expected no permissions for disabled binding")
	}

	// 5. 启用 Binding 并再次模拟
	_, _ = rbacservice.UpdateRoleBinding(ctx, rbID, &models.RoleBinding{
		ID:               rbID,
		Name:             "Test Binding",
		ServiceAccountID: saID,
		RoleIDs:          []string{roleID},
		Enabled:          true,
	})

	perms, err = rbacservice.SimulatePermissions(ctx, saID, "get", "dns")
	if err != nil {
		t.Fatalf("SimulatePermissions failed: %v", err)
	}
	
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

	// 5.5. 验证 ServiceAccount 启用/禁用
	if !sa.Enabled {
		t.Error("Expected ServiceAccount to be enabled by default")
	}
	sa.Enabled = false
	_, _ = rbacservice.UpdateServiceAccount(ctx, saID, sa)
	
	// 验证禁用后权限模拟应为空
	perms, _ = rbacservice.SimulatePermissions(ctx, saID, "get", "dns")
	if perms.AllowedAll || len(perms.AllowedInstances) > 0 {
		t.Error("Expected no permissions for disabled ServiceAccount")
	}

	sa.Enabled = true
	_, _ = rbacservice.UpdateServiceAccount(ctx, saID, sa)

	// 6. 重置 Token 验证
	oldToken := sa.Token
	resetSA, err := rbacservice.ResetServiceAccountToken(ctx, saID)
	if err != nil {
		t.Fatalf("Reset token failed: %v", err)
	}
	if resetSA.Token == oldToken {
		t.Error("Token should have changed after reset")
	}

	// 7. 级联删除验证: 删除 Role
	err = rbacservice.DeleteRole(ctx, roleID)
	if err != nil {
		t.Fatalf("DeleteRole failed: %v", err)
	}
	// RoleBinding 应该被删除 (因为它是唯一的 Role)
	rbResp, _ := rbacservice.ListRoleBindings(ctx, 1, 10, "")
	if rbResp.Total > 0 {
		t.Error("RoleBinding should be deleted after its only role is removed")
	}
}

func TestServiceAccountIDValidation(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := context.Background()

	invalidIDs := []string{"", "invalid id", "测试账号", "sa@123"}
	for _, id := range invalidIDs {
		_, err := rbacservice.CreateServiceAccount(ctx, &models.ServiceAccount{
			ID:   id,
			Name: "Invalid Test",
		})
		if err == nil {
			t.Errorf("Expected error for invalid SA ID '%s', but got nil", id)
		}
	}

	validIDs := []string{"sa-1", "SA_02", "123-abc"}
	for _, id := range validIDs {
		_, err := rbacservice.CreateServiceAccount(ctx, &models.ServiceAccount{
			ID:   id,
			Name: "Valid Test",
		})
		if err != nil {
			t.Errorf("Expected success for valid SA ID '%s', but got error: %v", id, err)
		}
	}
}
