package unit

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	authservice "homelab/pkg/services/auth"
	rbacservice "homelab/pkg/services/rbac"
	"homelab/tests"
	"testing"

	"github.com/golang-jwt/jwt/v5"
)

func TestRBACWildcardRobustness(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := context.Background()
	adminCtx := auth.WithPermissions(ctx, &models.ResourcePermissions{AllowedAll: true})

	// 1. Create SA
	_, _ = rbacservice.CreateServiceAccount(adminCtx, &models.ServiceAccount{ID: "worker", Name: "Worker"})

	// 2. Create Role with specific resource
	role, _ := rbacservice.CreateRole(adminCtx, &models.Role{
		Name: "DNS Manager",
		Rules: []models.PolicyRule{
			{Resource: "dns/example.com", Verbs: []string{"*"}},
		},
	})

	// 3. Bind
	_, _ = rbacservice.CreateRoleBinding(adminCtx, &models.RoleBinding{
		Name: "Bind", ServiceAccountID: "worker", RoleIDs: []string{role.ID}, Enabled: true,
	})

	// Test Case: Exact match
	perms, _ := authservice.GetPermissions(ctx, "worker", "get", "dns/example.com")
	if !perms.AllowedAll {
		t.Error("Should allow exact match")
	}

	// Test Case: Sub-resource match (Prefix)
	perms, _ = authservice.GetPermissions(ctx, "worker", "get", "dns/example.com/www/A")
	if !perms.AllowedAll {
		t.Error("Should allow sub-resource match via prefix")
	}

	// Test Case: Sibling resource (False positive check)
	// "dns/example.com" should NOT match "dns/example.com.cn"
	perms, _ = authservice.GetPermissions(ctx, "worker", "get", "dns/example.com.cn")
	if perms.AllowedAll {
		t.Error("Should NOT allow sibling resource match (potential prefix vulnerability)")
	}
}

func TestJWTForgeryRobustness(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	common.Opts.JWTSecret = "correct-secret"

	// 1. Wrong Secret
	wrongToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "root", "jti": "any",
	})
	wrongTokenStr, _ := wrongToken.SignedString([]byte("wrong-secret"))

	ok, _ := authservice.Verify(context.Background(), wrongTokenStr, "127.0.0.1", "UA")
	if ok {
		t.Error("Should reject token with wrong signature")
	}

	// 2. Wrong 'sub' type
	// Trying to use a Root token as an SA token
	rootToken, _ := authservice.Login(context.Background(), "admin", "", "127.0.0.1", "UA") // Use default admin password

	saID, _ := authservice.VerifySAToken(context.Background(), rootToken)
	if saID != "" {
		t.Error("Should not allow root token to be used as SA token")
	}

	// 3. Missing JTI for Root
	malformedRoot := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "root", // Missing jti
	})
	malformedRootStr, _ := malformedRoot.SignedString([]byte("correct-secret"))

	ok, _ = authservice.Verify(context.Background(), malformedRootStr, "127.0.0.1", "UA")
	if ok {
		t.Error("Should reject root token missing jti")
	}
}

func TestPaginationRobustness(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	adminCtx := auth.WithPermissions(context.Background(), &models.ResourcePermissions{AllowedAll: true})

	// Create 5 items
	for i := 0; i < 5; i++ {
		_, _ = rbacservice.CreateServiceAccount(adminCtx, &models.ServiceAccount{ID: "sa-" + string(rune('a'+i))})
	}

	// Test: Page out of range
	resp, _ := rbacservice.ListServiceAccounts(adminCtx, 10, 10, "")
	if resp.Total != 5 {
		t.Errorf("Total count should be 5, got %d", resp.Total)
	}
	// Items should be an empty slice, not nil or error
	if resp.Items == nil {
		t.Error("Items should be empty slice, not nil")
	}
}
