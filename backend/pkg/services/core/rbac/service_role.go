package rbac

import (
	"context"
	"errors"
	"fmt"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	rbacrepo "homelab/pkg/repositories/core/rbac"

	rbacmodel "homelab/pkg/models/core/rbac"
	"homelab/pkg/models/shared"

	"github.com/google/uuid"
)

func CreateRole(ctx context.Context, role *rbacmodel.Role) (*rbacmodel.Role, error) {
	role.ID = uuid.New().String()
	if err := normalizeRole(role); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	existing, _ := rbacrepo.GetCachedRole(ctx, role.ID)
	if existing != nil {
		return nil, errors.New("Role ID already exists")
	}

	err := rbacrepo.RoleRepo.Cow(ctx, role.ID, func(res *shared.Resource[rbacmodel.RoleV1Meta, rbacmodel.RoleV1Status]) error {
		res.Meta = role.Meta
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	})

	message := fmt.Sprintf("Created Role: %s (id: %s) with rules: %+v", role.Meta.Name, role.ID, role.Meta.Rules)
	if err != nil {
		commonaudit.FromContext(ctx).Log("CreateRole", role.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateRole", role.ID, message, true)

	updated, _ := rbacrepo.GetCachedRole(ctx, role.ID)
	return updated, nil
}

func UpdateRole(ctx context.Context, id string, role *rbacmodel.Role) (*rbacmodel.Role, error) {
	if err := normalizeRole(role); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	err := rbacrepo.RoleRepo.PatchMeta(ctx, id, role.Generation, func(m *rbacmodel.RoleV1Meta) {
		m.Name = role.Meta.Name
		m.Rules = role.Meta.Rules
	})

	message := fmt.Sprintf("Updated Role %s", id)
	if err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRole", id, message, false)
		return nil, err
	}

	updated, _ := rbacrepo.GetCachedRole(ctx, id)
	commonaudit.FromContext(ctx).Log("UpdateRole", id, message, true)
	return updated, nil
}

func DeleteRole(ctx context.Context, id string) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	release, err := lockRBAC(ctx, "global")
	if err != nil {
		return err
	}
	defer release()

	existing, err := rbacrepo.GetCachedRole(ctx, id)
	if err != nil {
		return errors.New("Role not found")
	}

	// Cascade update/delete RoleBindings
	rbs, err := rbacrepo.ScanAllRoleBindings(ctx)
	if err == nil {
		for _, rb := range rbs {
			newRoleIDs := make([]string, 0)
			found := false
			for _, rid := range rb.Meta.RoleIDs {
				if rid == id {
					found = true
				} else {
					newRoleIDs = append(newRoleIDs, rid)
				}
			}
			if found {
				if len(newRoleIDs) == 0 {
					_ = rbacrepo.BindingRepo.Delete(ctx, rb.ID)
				} else {
					rb.Meta.RoleIDs = newRoleIDs
					if err := rbacrepo.BindingRepo.Save(ctx, &rb); err == nil {
						rbacrepo.InvalidateCache("")
					}
				}
			}
		}
	}

	message := fmt.Sprintf("Deleted Role: %s (name: %s) with rules: %+v", existing.ID, existing.Meta.Name, existing.Meta.Rules)
	if err := rbacrepo.RoleRepo.Delete(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRole", id, message, false)
		return err
	}
	rbacrepo.InvalidateCache(id)
	commonaudit.FromContext(ctx).Log("DeleteRole", id, message, true)
	return nil
}
