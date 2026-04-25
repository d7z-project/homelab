package rbac

import (
	"context"
	"testing"

	"homelab/pkg/common"
	rbacmodel "homelab/pkg/models/core/rbac"

	"gopkg.d7z.net/middleware/kv"
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
	common.DB = db
	RoleRepo = common.NewBaseRepository[rbacmodel.RoleV1Meta, rbacmodel.RoleV1Status]("auth", "roles")
	ClearCache()

	if err := RoleRepo.Save(context.Background(), &rbacmodel.Role{
		ID:              "role-1",
		Meta:            rbacmodel.RoleV1Meta{Name: "before", Rules: []rbacmodel.PolicyRule{{Resource: "rbac", Verbs: []string{"get"}}}},
		Generation:      1,
		ResourceVersion: 0,
	}); err != nil {
		t.Fatalf("seed role: %v", err)
	}

	role, err := GetCachedRole(context.Background(), "role-1")
	if err != nil {
		t.Fatalf("get cached role: %v", err)
	}
	if role.Meta.Name != "before" {
		t.Fatalf("unexpected cached role: %#v", role)
	}

	if err := RoleRepo.PatchMeta(context.Background(), "role-1", 1, func(meta *rbacmodel.RoleV1Meta) {
		meta.Name = "after"
	}); err != nil {
		t.Fatalf("patch role: %v", err)
	}

	cachedRole, err := GetCachedRole(context.Background(), "role-1")
	if err != nil {
		t.Fatalf("get stale cached role: %v", err)
	}
	if cachedRole.Meta.Name != "before" {
		t.Fatalf("expected stale cache before invalidation, got %#v", cachedRole)
	}

	InvalidateCache("role-1")

	freshRole, err := GetCachedRole(context.Background(), "role-1")
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
	common.DB = db
	BindingRepo = common.NewBaseRepository[rbacmodel.RoleBindingV1Meta, rbacmodel.RoleBindingV1Status]("auth", "rolebindings")

	for _, id := range []string{"binding-a", "binding-b"} {
		bindingID := id
		if err := BindingRepo.Save(context.Background(), &rbacmodel.RoleBinding{
			ID:              bindingID,
			Meta:            rbacmodel.RoleBindingV1Meta{Name: bindingID, ServiceAccountID: "sa-1", RoleIDs: []string{"role-1"}, Enabled: true},
			Generation:      1,
			ResourceVersion: 0,
		}); err != nil {
			t.Fatalf("seed role binding %s: %v", bindingID, err)
		}
	}

	items, err := ScanAllRoleBindings(context.Background())
	if err != nil {
		t.Fatalf("scan all role bindings: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 role bindings, got %d", len(items))
	}
}
