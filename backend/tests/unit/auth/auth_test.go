package auth_test

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	authservice "homelab/pkg/services/auth"
	rbacservice "homelab/pkg/services/rbac"
	"homelab/tests"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestRootAuthAndSession(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := context.Background()
	common.Opts.RootPassword = "test-password"
	common.Opts.JWTSecret = "test-secret"

	// 1. Login
	token, err := authservice.Login(ctx, "test-password", "", "127.0.0.1", "Mozilla/5.0")
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	// 2. Verify Success
	ok, err := authservice.Verify(ctx, token, "127.0.0.1", "Mozilla/5.0")
	if err != nil || !ok {
		t.Errorf("Verification failed: ok=%v, err=%v", ok, err)
	}

	// 3. Verify Failure (IP Mismatch)
	ok, _ = authservice.Verify(ctx, token, "192.168.1.1", "Mozilla/5.0")
	if ok {
		t.Error("Expected verification failure for IP mismatch")
	}

	// 4. List Sessions
	rootCtx := auth.WithAuth(ctx, &auth.AuthContext{Type: "root"})
	sessions, err := authservice.ScanSessions(rootCtx)
	if err != nil {
		t.Fatalf("ScanSessions failed: %v", err)
	}
	if len(sessions) != 1 {
		t.Errorf("Expected 1 session, got %d", len(sessions))
	}

	// 5. Revoke Session
	sessionID := sessions[0].ID
	err = authservice.RevokeSession(rootCtx, sessionID)
	if err != nil {
		t.Fatalf("RevokeSession failed: %v", err)
	}

	// 6. Verify after revocation (should fail)
	ok, _ = authservice.Verify(ctx, token, "127.0.0.1", "Mozilla/5.0")
	if ok {
		t.Error("Expected verification failure after revocation")
	}
}

func TestServiceAccountAuth(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := context.Background()
	adminCtx := auth.WithPermissions(ctx, &models.ResourcePermissions{AllowedInstances: []string{"rbac"}})
	common.Opts.JWTSecret = "test-secret"

	// 1. Create SA
	sa, err := rbacservice.CreateServiceAccount(adminCtx, &models.ServiceAccount{ID: "test-sa", Meta: models.ServiceAccountV1Meta{Name: "Test SA",
	}})
	if err != nil {
		t.Fatalf("CreateServiceAccount failed: %v", err)
	}
	token := sa.Meta.Token

	// 2. Verify SA Token
	saID, err := authservice.VerifySAToken(ctx, token)
	if err != nil || saID != "test-sa" {
		t.Errorf("VerifySAToken failed: saID=%s, err=%v", saID, err)
	}

	// 3. IsSAEnabled check
	if !authservice.IsSAEnabled(ctx, "test-sa", token) {
		t.Error("Expected SA to be enabled and token to match")
	}

	// 4. Reset Token
	newSA, err := rbacservice.ResetServiceAccountToken(adminCtx, "test-sa")
	if err != nil {
		t.Fatalf("ResetServiceAccountToken failed: %v", err)
	}
	newToken := newSA.Meta.Token

	// 5. Verify old token (should fail matching)
	if authservice.IsSAEnabled(ctx, "test-sa", token) {
		t.Error("Old token should be invalid after reset")
	}

	// 6. Verify new token
	if !authservice.IsSAEnabled(ctx, "test-sa", newToken) {
		t.Error("New token should be valid")
	}

	// 7. Disable SA
	newSA.Meta.Enabled = false
	_, err = rbacservice.UpdateServiceAccount(adminCtx, "test-sa", newSA)
	if err != nil {
		t.Fatalf("UpdateServiceAccount failed: %v", err)
	}

	if authservice.IsSAEnabled(ctx, "test-sa", newToken) {
		t.Error("Token should be invalid when SA is disabled")
	}
}

func TestJWTClaims(t *testing.T) {
	common.Opts.JWTSecret = "test-secret"

	// Test SA token claims
	tokenStr, _ := authservice.CreateSAToken("my-sa")
	token, _ := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte("test-secret"), nil
	})

	claims := token.Claims.(jwt.MapClaims)
	if claims["sub"] != "sa" || claims["sa_id"] != "my-sa" {
		t.Errorf("Invalid SA claims: %v", claims)
	}
	if _, ok := claims["exp"]; ok {
		t.Error("SA token should not have exp claim")
	}
}

func TestSessionSlidingExpiration(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := context.Background()
	common.Opts.RootPassword = "pass"
	common.Opts.JWTSecret = "sec"
	common.Opts.SessionTTL = "100ms" // Short TTL for test

	// 1. Login
	token, _ := authservice.Login(ctx, "pass", "", "127.0.0.1", "UA")

	// 2. Immediate verify (should success and refresh)
	ok, _ := authservice.Verify(ctx, token, "127.0.0.1", "UA")
	if !ok {
		t.Fatal("Initial verify should succeed")
	}

	// 3. Wait for 60ms (less than 100ms) and verify again (should refresh)
	time.Sleep(60 * time.Millisecond)
	ok, _ = authservice.Verify(ctx, token, "127.0.0.1", "UA")
	if !ok {
		t.Error("Verify after 60ms should succeed (sliding)")
	}

	// 4. Wait for 120ms (more than 100ms) without verify
	time.Sleep(120 * time.Millisecond)
	ok, _ = authservice.Verify(ctx, token, "127.0.0.1", "UA")
	if ok {
		t.Error("Verify after TTL should fail")
	}
}
