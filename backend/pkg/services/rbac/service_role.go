package rbac

import (
	"context"
	"errors"
	"fmt"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	rbacrepo "homelab/pkg/repositories/rbac"
	"strings"

	"github.com/google/uuid"
)

func CreateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	if role.ID == "" {
		role.ID = uuid.New().String()
	}
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

	message := fmt.Sprintf("Created Role: %s (id: %s) with rules: %+v", role.Name, role.ID, role.Rules)
	if err := rbacrepo.SaveRole(ctx, role); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRole", role.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateRole", role.ID, message, true)
	return role, nil
}

func UpdateRole(ctx context.Context, id string, role *models.Role) (*models.Role, error) {
	if err := role.Bind(nil); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	release, err := lockRBAC(ctx, "role:"+id)
	if err != nil {
		return nil, err
	}
	defer release()

	if role.ID != id {
		return nil, errors.New("id in body does not match path")
	}

	existing, err := rbacrepo.GetRole(ctx, id)
	if err != nil {
		return nil, errors.New("Role not found")
	}

	role.ID = id
	changes := []string{}
	if existing.Name != role.Name {
		changes = append(changes, fmt.Sprintf("name: '%s' -> '%s'", existing.Name, role.Name))
	}
	changes = append(changes, fmt.Sprintf("rules updated: %+v -> %+v", existing.Rules, role.Rules))

	message := fmt.Sprintf("Updated Role: %s: %s", role.ID, strings.Join(changes, ", "))
	if err := rbacrepo.SaveRole(ctx, role); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRole", role.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateRole", role.ID, message, true)
	return role, nil
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
			for _, rid := range rb.RoleIDs {
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
					rb.RoleIDs = newRoleIDs
					rbacrepo.SaveRoleBinding(ctx, &rb)
				}
			}
		}
	}

	message := fmt.Sprintf("Deleted Role: %s (name: %s) with rules: %+v", existing.ID, existing.Name, existing.Rules)
	if err := rbacrepo.DeleteRole(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRole", id, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteRole", id, message, true)
	return nil
}
