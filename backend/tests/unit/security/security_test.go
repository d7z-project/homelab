package security_test

import (
	"context"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsservice "homelab/pkg/services/dns"
	rbacservice "homelab/pkg/services/rbac"
	"homelab/tests"
	"testing"
)

func TestDNSListPermissionFiltering(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// Admin context to setup data
	adminCtx := auth.WithPermissions(context.Background(), &models.ResourcePermissions{AllowedAll: true})

	// Create two domains
	_, _ = dnsservice.CreateDomain(adminCtx, &models.Domain{Meta: models.DomainV1Meta{Name: "public.com"}})
	_, _ = dnsservice.CreateDomain(adminCtx, &models.Domain{Meta: models.DomainV1Meta{Name: "private.com"}})

	// Context with permission only for public.com
	userCtx := auth.WithPermissions(context.Background(), &models.ResourcePermissions{
		AllowedInstances: []string{"network/dns/public.com"},
	})

	// List domains
	res, err := dnsservice.ScanDomains(userCtx, "", 10, "")
	if err != nil {
		t.Fatalf("ScanDomains failed: %v", err)
	}

	if len(res.Items) != 1 {
		t.Errorf("Expected 1 domain in filtered list, got %d", len(res.Items))
	}

	foundPublic := false
	for _, d := range res.Items {
		if d.Meta.Name == "private.com" {
			t.Error("User saw private.com without permission")
		}
		if d.Meta.Name == "public.com" {
			foundPublic = true
		}
	}
	if !foundPublic {
		t.Error("User did not see public.com with permission")
	}
}

func TestDNSExportPermissionFiltering(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	adminCtx := auth.WithPermissions(context.Background(), &models.ResourcePermissions{AllowedAll: true})

	_, _ = dnsservice.CreateDomain(adminCtx, &models.Domain{Meta: models.DomainV1Meta{Name: "public.com", Enabled: true}})
	_, _ = dnsservice.CreateDomain(adminCtx, &models.Domain{Meta: models.DomainV1Meta{Name: "private.com", Enabled: true}})

	userCtx := auth.WithPermissions(context.Background(), &models.ResourcePermissions{
		AllowedInstances: []string{"network/dns/public.com"},
	})

	export, err := dnsservice.ExportAll(userCtx)
	if err != nil {
		t.Fatalf("ExportAll failed: %v", err)
	}

	if len(export.Domains) != 1 {
		t.Errorf("Expected 1 domain in export, got %d", len(export.Domains))
	}
	if export.Domains[0].Name != "public.com" {
		t.Errorf("Expected public.com in export, got %s", export.Domains[0].Name)
	}
}

func TestRBACServicePermissionChecks(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// Context without rbac permission
	userCtx := auth.WithPermissions(context.Background(), &models.ResourcePermissions{AllowedAll: false})

	// Try to list service accounts
	_, err := rbacservice.ScanServiceAccounts(userCtx, "", 10, "")
	if err == nil {
		t.Error("Expected error when listing SAs without rbac permission")
	}

	// Try to create a role
	_, err = rbacservice.CreateRole(userCtx, &models.Role{Name: "Test"})
	if err == nil {
		t.Error("Expected error when creating role without rbac permission")
	}

	// Context with rbac permission
	adminCtx := auth.WithPermissions(context.Background(), &models.ResourcePermissions{
		AllowedInstances: []string{"rbac"},
	})

	_, err = rbacservice.ScanServiceAccounts(adminCtx, "", 10, "")
	if err != nil {
		t.Errorf("Expected success with rbac permission, got error: %v", err)
	}
}
