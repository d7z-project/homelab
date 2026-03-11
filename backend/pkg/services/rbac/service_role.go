package rbac

import (
	"context"
	"errors"
	"fmt"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	rbacrepo "homelab/pkg/repositories/rbac"

	"github.com/google/uuid"
)

func CreateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	role.ID = uuid.New().String()
	if err := role.Bind(nil); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	existing, _ := rbacrepo.GetRole(ctx, role.ID)
	if existing != nil {
		return nil, errors.New("Role ID already exists")
	}

	err := rbacrepo.RoleRepo.Cow(ctx, role.ID, func(res *models.Resource[models.RoleV1Meta, models.RoleV1Status]) error {
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

	updated, _ := rbacrepo.GetRole(ctx, role.ID)
	return updated, nil
}

func UpdateRole(ctx context.Context, id string, role *models.Role) (*models.Role, error) {
	if err := role.Bind(nil); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	err := rbacrepo.RoleRepo.PatchMeta(ctx, id, role.Generation, func(m *models.RoleV1Meta) {
		m.Name = role.Meta.Name
		m.Rules = role.Meta.Rules
	})

	message := fmt.Sprintf("Updated Role %s", id)
	if err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRole", id, message, false)
		return nil, err
	}

	updated, _ := rbacrepo.GetRole(ctx, id)
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

	existing, err := rbacrepo.GetRole(ctx, id)
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
					rbacrepo.DeleteRoleBinding(ctx, rb.ID)
				} else {
					rb.Meta.RoleIDs = newRoleIDs
					rbacrepo.SaveRoleBinding(ctx, &rb)
				}
			}
		}
	}

	message := fmt.Sprintf("Deleted Role: %s (name: %s) with rules: %+v", existing.ID, existing.Meta.Name, existing.Meta.Rules)
	if err := rbacrepo.DeleteRole(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRole", id, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteRole", id, message, true)
	return nil
}
