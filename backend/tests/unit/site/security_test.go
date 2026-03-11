package site_test

import (
	"context"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	authservice "homelab/pkg/services/auth"
	"homelab/pkg/services/rbac"
	"homelab/pkg/services/site"
	"homelab/tests"
	"strings"
	"testing"
)

func TestSiteSecurity(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctxRoot := tests.SetupMockRootContext()

	// 1. Create a ServiceAccount
	sa, err := rbac.CreateServiceAccount(ctxRoot, &models.ServiceAccount{ID: "site-tester", Meta: models.ServiceAccountV1Meta{Name: "Site Tester"}})
	if err != nil {
		t.Fatalf("Failed to create SA: %v", err)
	}

	analysis := site.NewAnalysisEngine(nil)
	service := site.NewSitePoolService(analysis, nil)

	// 2. Setup Site Pools
	group1 := &models.SiteGroup{ID: "site-pool-1", Meta: models.SiteGroupV1Meta{Name: "Pool 1"}}
	group2 := &models.SiteGroup{ID: "site-pool-2", Meta: models.SiteGroupV1Meta{Name: "Pool 2"}}

	_ = service.CreateGroup(ctxRoot, group1)
	_ = service.CreateGroup(ctxRoot, group2)

	// 3. Setup permissions for SA: Only allowed to manage pool-1
	role := &models.Role{ID: "site-role", Meta: models.RoleV1Meta{Name: "Site Manager",
		Rules: []models.PolicyRule{
			{Resource: "network/site/" + group1.ID, Verbs: []string{"*"}},
			{Resource: "network/site", Verbs: []string{"list", "get"}},
		},
	}}
	_, _ = rbac.CreateRole(ctxRoot, role)
	_, _ = rbac.CreateRoleBinding(ctxRoot, &models.RoleBinding{ID: "site-binding", Meta: models.RoleBindingV1Meta{
		Name:             "Site Binding",
		ServiceAccountID: sa.ID,
		RoleIDs:          []string{role.ID},
		Enabled:          true,
	}})

	// Create a context impersonating this SA
	perms, _ := authservice.GetPermissions(context.Background(), sa.ID, "delete", "network/site/"+group1.ID)
	ctxSA := auth.WithAuth(context.Background(), &auth.AuthContext{ID: sa.ID, Type: "sa"})
	ctxSA = auth.WithPermissions(ctxSA, perms)

	t.Run("Allow delete authorized site pool", func(t *testing.T) {
		err := service.DeleteGroup(ctxSA, group1.ID)
		if err != nil {
			t.Errorf("Should allow deleting pool-1: %v", err)
		}
	})

	t.Run("Deny delete unauthorized site pool", func(t *testing.T) {
		perms2, _ := authservice.GetPermissions(context.Background(), sa.ID, "delete", "network/site/"+group2.ID)
		ctxSA2 := auth.WithPermissions(ctxSA, perms2)

		err := service.DeleteGroup(ctxSA2, group2.ID)
		if err == nil {
			t.Error("Should NOT allow deleting pool-2")
		} else if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("Expected permission denied error, got: %v", err)
		}
	})
}
