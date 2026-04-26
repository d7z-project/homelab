package rbac

import (
	"context"
	"errors"
	"fmt"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	rbacrepo "homelab/pkg/repositories/core/rbac"

	rbacmodel "homelab/pkg/models/core/rbac"

	"github.com/google/uuid"
)

func CreateRoleBinding(ctx context.Context, rb *rbacmodel.RoleBinding) (*rbacmodel.RoleBinding, error) {
	if err := normalizeRoleBinding(rb); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	// Verify ServiceAccount exists
	if _, err := rbacrepo.GetServiceAccount(ctx, rb.Meta.ServiceAccountID); err != nil {
		return nil, fmt.Errorf("service account '%s' not found", rb.Meta.ServiceAccountID)
	}

	// Verify all Roles exist
	for _, rid := range rb.Meta.RoleIDs {
		if _, err := rbacrepo.GetCachedRole(ctx, rid); err != nil {
			return nil, fmt.Errorf("role '%s' not found", rid)
		}
	}

	rb.ID = uuid.New().String()

	err := rbacrepo.SaveRoleBinding(ctx, rb)

	message := fmt.Sprintf("Created RoleBinding: %s (id: %s, SA: %s, Roles: %v, enabled: %v)",
		rb.Meta.Name, rb.ID, rb.Meta.ServiceAccountID, rb.Meta.RoleIDs, rb.Meta.Enabled)
	if err != nil {
		commonaudit.FromContext(ctx).Log("CreateRoleBinding", rb.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateRoleBinding", rb.ID, message, true)

	updated, _ := rbacrepo.GetRoleBinding(ctx, rb.ID)
	return updated, nil
}

func UpdateRoleBinding(ctx context.Context, id string, rb *rbacmodel.RoleBinding) (*rbacmodel.RoleBinding, error) {
	if err := normalizeRoleBinding(rb); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	// Verify ServiceAccount exists
	if _, err := rbacrepo.GetServiceAccount(ctx, rb.Meta.ServiceAccountID); err != nil {
		return nil, fmt.Errorf("service account '%s' not found", rb.Meta.ServiceAccountID)
	}

	// Verify all Roles exist
	for _, rid := range rb.Meta.RoleIDs {
		if _, err := rbacrepo.GetCachedRole(ctx, rid); err != nil {
			return nil, fmt.Errorf("role '%s' not found", rid)
		}
	}

	existing, err := rbacrepo.GetRoleBinding(ctx, id)
	if err != nil {
		return nil, err
	}
	existing.Meta = rb.Meta
	err = rbacrepo.SaveRoleBinding(ctx, existing)

	message := fmt.Sprintf("Updated RoleBinding %s", id)
	if err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRoleBinding", id, message, false)
		return nil, err
	}

	updated, _ := rbacrepo.GetRoleBinding(ctx, id)
	commonaudit.FromContext(ctx).Log("UpdateRoleBinding", id, message, true)
	return updated, nil
}

func DeleteRoleBinding(ctx context.Context, id string) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	existing, err := rbacrepo.GetRoleBinding(ctx, id)
	if err != nil {
		return errors.New("RoleBinding not found")
	}

	message := fmt.Sprintf("Deleted RoleBinding: %s (name: %s, SA: %s, Roles: %v, enabled: %v)",
		existing.ID, existing.Meta.Name, existing.Meta.ServiceAccountID, existing.Meta.RoleIDs, existing.Meta.Enabled)
	if err := rbacrepo.DeleteRoleBinding(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRoleBinding", id, message, false)
		return err
	}
	rbacrepo.InvalidateCache("")
	commonaudit.FromContext(ctx).Log("DeleteRoleBinding", id, message, true)
	return nil
}
