package common_test

import (
	"context"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	rbacservice "homelab/pkg/services/rbac"
	"homelab/tests"
	"testing"
)

func TestRBACConsistency(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := context.Background()
	adminCtx := auth.WithPermissions(ctx, &models.ResourcePermissions{AllowedAll: true})

	// 0. Create ServiceAccount
	_, err := rbacservice.CreateServiceAccount(adminCtx, &models.ServiceAccount{ID: "test-sa", Meta: models.ServiceAccountV1Meta{Name: "Test SA"}})
	if err != nil {
		t.Fatalf("CreateServiceAccount failed: %v", err)
	}

	// 1. Create a Role and Binding
	role, err := rbacservice.CreateRole(adminCtx, &models.Role{Meta: models.RoleV1Meta{Name: "Test Role", Rules: []models.PolicyRule{{Resource: "network/dns", Verbs: []string{"*"}}}}})
	if err != nil {
		t.Fatalf("CreateRole failed: %v", err)
	}

	_, err = rbacservice.CreateRoleBinding(adminCtx, &models.RoleBinding{ID: "test-binding", Meta: models.RoleBindingV1Meta{
		Name:             "Test Binding",
		RoleIDs:          []string{role.ID},
		ServiceAccountID: "test-sa",
		Enabled:          true,
	}})
	if err != nil {
		t.Fatalf("CreateRoleBinding failed: %v", err)
	}

	// 2. Verify it can be listed
	resp, err := rbacservice.ScanRoleBindings(adminCtx, "", 10, "")
	if err != nil {
		t.Fatalf("ScanRoleBindings failed: %v", err)
	}
	if len(resp.Items) != 1 {
		t.Errorf("Expected 1 role binding, got %d", len(resp.Items))
	}

	// 3. Create another binding for the same SA
	_, err = rbacservice.CreateRoleBinding(adminCtx, &models.RoleBinding{ID: "test-binding-2", Meta: models.RoleBindingV1Meta{
		Name:             "Test Binding 2",
		RoleIDs:          []string{role.ID},
		ServiceAccountID: "test-sa",
		Enabled:          true,
	}})
	if err != nil {
		t.Fatalf("CreateRoleBinding 2 failed: %v", err)
	}

	// 4. Verify ServiceAccount deletion clears related RoleBindings
	err = rbacservice.DeleteServiceAccount(adminCtx, "test-sa")
	if err != nil {
		t.Fatalf("DeleteServiceAccount failed: %v", err)
	}

	resp, _ = rbacservice.ScanRoleBindings(adminCtx, "", 10, "")
	if len(resp.Items) != 0 {
		t.Errorf("RoleBindings should be deleted after ServiceAccount deletion, still have %d", len(resp.Items))
	}
}
