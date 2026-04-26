package rbac

import (
	"context"
	"testing"

	"gopkg.d7z.net/middleware/kv"
	"homelab/pkg/common"
	rbacmodel "homelab/pkg/models/core/rbac"
)

func TestGetCachedRoleAndInvalidateCache(t *testing.T) {
	t.Parallel()

	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	roleRepo = common.NewResourceRepository[rbacmodel.RoleV1Meta, rbacmodel.RoleV1Status](db, "auth", "roles")
	ClearCache()
	ctx := context.Background()

	if err := roleRepo.Save(ctx, &rbacmodel.Role{
		ID:              "role-1",
		Meta:            rbacmodel.RoleV1Meta{Name: "before", Rules: []rbacmodel.PolicyRule{{Resource: "rbac", Verbs: []string{"get"}}}},
		Generation:      1,
		ResourceVersion: 0,
	}); err != nil {
		t.Fatalf("seed role: %v", err)
	}

	role, err := GetCachedRole(ctx, "role-1")
	if err != nil {
		t.Fatalf("get cached role: %v", err)
	}
	if role.Meta.Name != "before" {
		t.Fatalf("unexpected cached role: %#v", role)
	}

	role, err = GetRole(ctx, "role-1")
	if err != nil {
		t.Fatalf("get role for update: %v", err)
	}
	role.Meta.Name = "after"
	if err := SaveRole(ctx, role); err != nil {
		t.Fatalf("save role: %v", err)
	}

	cachedRole, err := GetCachedRole(ctx, "role-1")
	if err != nil {
		t.Fatalf("get stale cached role: %v", err)
	}
	if cachedRole.Meta.Name != "before" {
		t.Fatalf("expected stale cache before invalidation, got %#v", cachedRole)
	}

	InvalidateCache("role-1")

	freshRole, err := GetCachedRole(ctx, "role-1")
	if err != nil {
		t.Fatalf("get fresh cached role: %v", err)
	}
	if freshRole.Meta.Name != "after" {
		t.Fatalf("expected refreshed cache after invalidation, got %#v", freshRole)
	}
}

func TestScanAllRoleBindings(t *testing.T) {
	t.Parallel()

	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	bindingRepo = common.NewResourceRepository[rbacmodel.RoleBindingV1Meta, rbacmodel.RoleBindingV1Status](db, "auth", "rolebindings")
	ctx := context.Background()

	for _, id := range []string{"binding-a", "binding-b"} {
		bindingID := id
		if err := bindingRepo.Save(ctx, &rbacmodel.RoleBinding{
			ID:              bindingID,
			Meta:            rbacmodel.RoleBindingV1Meta{Name: bindingID, ServiceAccountID: "sa-1", RoleIDs: []string{"role-1"}, Enabled: true},
			Generation:      1,
			ResourceVersion: 0,
		}); err != nil {
			t.Fatalf("seed role binding %s: %v", bindingID, err)
		}
	}

	items, err := ScanAllRoleBindings(ctx)
	if err != nil {
		t.Fatalf("scan all role bindings: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 role bindings, got %d", len(items))
	}
}
