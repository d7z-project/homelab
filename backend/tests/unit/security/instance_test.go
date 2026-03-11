package security_test

import (
	"context"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
	authservice "homelab/pkg/services/auth"
	"homelab/pkg/services/ip"
	"homelab/pkg/services/rbac"
	"homelab/pkg/services/site"
	"homelab/tests"
	"strings"
	"testing"
)

func TestIPPoolSecurityInstanceLevel(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctxRoot := tests.SetupMockRootContext()

	// 1. Create a ServiceAccount
	sa, err := rbac.CreateServiceAccount(ctxRoot, &models.ServiceAccount{ID: "ip-security-tester", Meta: models.ServiceAccountV1Meta{Name: "IP Security Tester"}})
	if err != nil {
		t.Fatalf("Failed to create SA: %v", err)
	}

	mmdb := ip.NewMMDBManager(nil)
	ae := ip.NewAnalysisEngine(mmdb)
	em := ip.NewExportManager(ae)
	service := ip.NewIPPoolService(ae, em)

	// 2. Setup IP Pools
	group1 := &models.IPPool{ID: "pool-1", Meta: models.IPPoolV1Meta{Name: "Pool 1"}}
	group2 := &models.IPPool{ID: "pool-2", Meta: models.IPPoolV1Meta{Name: "Pool 2"}}

	err = service.CreateGroup(ctxRoot, group1)
	if err != nil {
		t.Fatalf("Failed to create pool-1: %v", err)
	}
	err = service.CreateGroup(ctxRoot, group2)
	if err != nil {
		t.Fatalf("Failed to create pool-2: %v", err)
	}

	// 3. Setup permissions for SA: Only allowed to manage pool-1
	role := &models.Role{ID: "pool1-manager", Meta: models.RoleV1Meta{Name: "Pool1 Manager",
		Rules: []models.PolicyRule{
			{Resource: "network/ip/" + group1.ID, Verbs: []string{"*"}},
			{Resource: "network/ip", Verbs: []string{"list", "get"}}, // Global read
		},
	}}
	_, _ = rbac.CreateRole(ctxRoot, role)
	_, _ = rbac.CreateRoleBinding(ctxRoot, &models.RoleBinding{ID: "binding-pool1", Meta: models.RoleBindingV1Meta{
		Name:             "Binding Pool1",
		ServiceAccountID: sa.ID,
		RoleIDs:          []string{role.ID},
		Enabled:          true,
	}})

	// Create a context impersonating this SA
	perms, _ := authservice.GetPermissions(context.Background(), sa.ID, "delete", "network/ip/"+group1.ID)
	ctxSA := auth.WithAuth(context.Background(), &auth.AuthContext{ID: sa.ID, Type: "sa"})
	ctxSA = auth.WithPermissions(ctxSA, perms)

	t.Run("Allow delete authorized pool instance", func(t *testing.T) {
		err := service.DeleteGroup(ctxSA, group1.ID)
		if err != nil {
			t.Errorf("Should allow deleting pool-1: %v", err)
		}
	})

	t.Run("Deny delete unauthorized pool instance", func(t *testing.T) {
		// Re-evaluate permissions for pool-2
		perms2, _ := authservice.GetPermissions(context.Background(), sa.ID, "delete", "network/ip/"+group2.ID)
		ctxSA2 := auth.WithPermissions(ctxSA, perms2)

		err := service.DeleteGroup(ctxSA2, group2.ID)
		if err == nil {
			t.Error("Should NOT allow deleting pool-2")
		} else if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("Expected permission denied error, got: %v", err)
		} else {
			t.Logf("Correctly denied: %v", err)
		}
	})
}

func TestSitePoolSecurityInstanceLevel(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctxRoot := tests.SetupMockRootContext()

	// 1. Create a ServiceAccount
	sa, err := rbac.CreateServiceAccount(ctxRoot, &models.ServiceAccount{ID: "site-security-tester", Meta: models.ServiceAccountV1Meta{Name: "Site Security Tester"}})
	if err != nil {
		t.Fatalf("Failed to create SA: %v", err)
	}

	analysis := site.NewAnalysisEngine(nil)
	service := site.NewSitePoolService(analysis, nil)

	// 2. Setup Site Pools
	group1 := &models.SiteGroup{Name: "Pool 1"}
	group2 := &models.SiteGroup{Name: "Pool 2"}

	err = service.CreateGroup(ctxRoot, group1)
	if err != nil {
		t.Fatalf("Failed to create pool-1: %v", err)
	}
	err = service.CreateGroup(ctxRoot, group2)
	if err != nil {
		t.Fatalf("Failed to create pool-2: %v", err)
	}

	// 3. Setup permissions for SA: Only allowed to manage pool-1
	role := &models.Role{ID: "site-pool1-manager", Meta: models.RoleV1Meta{Name: "Site Pool1 Manager",
		Rules: []models.PolicyRule{
			{Resource: "network/site/" + group1.ID, Verbs: []string{"*"}},
			{Resource: "network/site", Verbs: []string{"list", "get"}}, // Global read
		},
	}}
	_, _ = rbac.CreateRole(ctxRoot, role)
	_, _ = rbac.CreateRoleBinding(ctxRoot, &models.RoleBinding{ID: "site-binding-pool1", Meta: models.RoleBindingV1Meta{
		Name:             "Site Binding Pool1",
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

func TestSecurityInstanceLevel(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctxRoot := tests.SetupMockRootContext()

	// 1. Create a ServiceAccount and verify hashing
	sa, err := rbac.CreateServiceAccount(ctxRoot, &models.ServiceAccount{ID: "security-tester", Meta: models.ServiceAccountV1Meta{Name: "Security Tester"}})
	if err != nil {
		t.Fatalf("Failed to create SA: %v", err)
	}

	plainToken := sa.Meta.Token // This is the plain JWT returned once
	if plainToken == "" {
		t.Fatal("Expected plain token in response, got empty")
	}

	// Verify that IsSAEnabled works with the plain token (it hashes internally)
	if !authservice.IsSAEnabled(context.Background(), sa.ID, plainToken) {
		t.Error("IsSAEnabled failed with plain token")
	}

	// 2. Setup Workflows
	wf1 := &models.Workflow{Name: "Workflow 1", ServiceAccountID: sa.ID, Steps: []models.Step{{ID: "s1", Type: "core/logger", Params: map[string]string{"message": "hi"}}}}
	wf2 := &models.Workflow{Name: "Workflow 2", ServiceAccountID: sa.ID, Steps: []models.Step{{ID: "s1", Type: "core/logger", Params: map[string]string{"message": "hi"}}}}

	wf1, err = actions.CreateWorkflow(ctxRoot, wf1)
	if err != nil {
		t.Fatalf("Failed to create wf1: %v", err)
	}
	wf2, err = actions.CreateWorkflow(ctxRoot, wf2)
	if err != nil {
		t.Fatalf("Failed to create wf2: %v", err)
	}

	// 3. Setup permissions for SA: Only allowed to manage wf-1
	role := &models.Role{ID: "wf1-manager", Meta: models.RoleV1Meta{Name: "WF1 Manager",
		Rules: []models.PolicyRule{
			{Resource: "actions/" + wf1.ID, Verbs: []string{"*"}},
			{Resource: "actions", Verbs: []string{"list", "get"}}, // Global read
		},
	}}
	_, _ = rbac.CreateRole(ctxRoot, role)
	_, _ = rbac.CreateRoleBinding(ctxRoot, &models.RoleBinding{ID: "binding-1", Meta: models.RoleBindingV1Meta{
		Name:             "Binding 1",
		ServiceAccountID: sa.ID,
		RoleIDs:          []string{role.ID},
		Enabled:          true,
	}})

	// Create a context impersonating this SA
	perms, _ := authservice.GetPermissions(context.Background(), sa.ID, "delete", "actions/"+wf1.ID)
	ctxSA := auth.WithAuth(context.Background(), &auth.AuthContext{ID: sa.ID, Type: "sa"})
	ctxSA = auth.WithPermissions(ctxSA, perms)

	t.Run("Allow delete authorized instance", func(t *testing.T) {
		err := actions.DeleteWorkflow(ctxSA, wf1.ID)
		if err != nil {
			t.Errorf("Should allow deleting wf-1: %v", err)
		}
	})

	t.Run("Deny delete unauthorized instance", func(t *testing.T) {
		// Re-evaluate permissions for wf-2
		perms2, _ := authservice.GetPermissions(context.Background(), sa.ID, "delete", "actions/"+wf2.ID)
		ctxSA2 := auth.WithPermissions(ctxSA, perms2)

		err := actions.DeleteWorkflow(ctxSA2, wf2.ID)
		if err == nil {
			t.Error("Should NOT allow deleting wf-2")
		} else if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("Expected permission denied error, got: %v", err)
		} else {
			t.Logf("Correctly denied: %v", err)
		}
	})

	t.Run("Deny run unauthorized instance", func(t *testing.T) {
		// Even if the user has DELETE permission on wf-1, they need EXECUTE on wf-2
		perms2, _ := authservice.GetPermissions(context.Background(), sa.ID, "execute", "actions/"+wf2.ID)
		ctxSA2 := auth.WithPermissions(ctxSA, perms2)

		_, err := actions.RunWorkflow(ctxSA2, wf2.ID, nil, "Manual")
		if err == nil {
			t.Error("Should NOT allow running wf-2")
		} else if !strings.Contains(err.Error(), "permission denied") {
			t.Errorf("Expected permission denied error, got: %v", err)
		} else {
			t.Logf("Correctly denied: %v", err)
		}
	})
}
