package rbac

import (
	"context"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	rbacrepo "homelab/pkg/repositories/rbac"
	"strings"

	"github.com/google/uuid"
)

func ListRoleBindings(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}
	rbs, total, err := rbacrepo.ListRoleBindings(ctx, uint64(page-1), uint(pageSize), search)
	if err != nil {
		return nil, err
	}

	var items []interface{}
	for _, rb := range rbs {
		items = append(items, rb)
	}

	return &common.PaginatedResponse{
		Items: items,
		Total: int(total),
		Page:  page,
	}, nil
}

func CreateRoleBinding(ctx context.Context, rb *models.RoleBinding) (*models.RoleBinding, error) {
	if err := rb.Bind(nil); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	// Verify ServiceAccount exists
	if _, err := rbacrepo.GetServiceAccount(ctx, rb.ServiceAccountID); err != nil {
		return nil, fmt.Errorf("service account '%s' not found", rb.ServiceAccountID)
	}

	// Verify all Roles exist
	for _, rid := range rb.RoleIDs {
		if _, err := rbacrepo.GetRole(ctx, rid); err != nil {
			return nil, fmt.Errorf("role '%s' not found", rid)
		}
	}

	if rb.ID == "" {
		rb.ID = uuid.New().String()
	}

	existing, _ := rbacrepo.GetRoleBinding(ctx, rb.ID)
	if existing != nil {
		return nil, errors.New("RoleBinding ID already exists")
	}

	message := fmt.Sprintf("Created RoleBinding: %s (id: %s, SA: %s, Roles: %v, enabled: %v)",
		rb.Name, rb.ID, rb.ServiceAccountID, rb.RoleIDs, rb.Enabled)
	if err := rbacrepo.SaveRoleBinding(ctx, rb); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRoleBinding", rb.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateRoleBinding", rb.ID, message, true)
	return rb, nil
}

func UpdateRoleBinding(ctx context.Context, id string, rb *models.RoleBinding) (*models.RoleBinding, error) {
	if err := rb.Bind(nil); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}
	if rb.ID != id {
		return nil, errors.New("id in body does not match path")
	}

	existing, err := rbacrepo.GetRoleBinding(ctx, id)
	if err != nil {
		return nil, errors.New("RoleBinding not found")
	}

	// Verify ServiceAccount exists
	if _, err := rbacrepo.GetServiceAccount(ctx, rb.ServiceAccountID); err != nil {
		return nil, fmt.Errorf("service account '%s' not found", rb.ServiceAccountID)
	}

	// Verify all Roles exist
	for _, rid := range rb.RoleIDs {
		if _, err := rbacrepo.GetRole(ctx, rid); err != nil {
			return nil, fmt.Errorf("role '%s' not found", rid)
		}
	}

	rb.ID = id
	changes := []string{}
	if existing.Name != rb.Name {
		changes = append(changes, fmt.Sprintf("name: '%s' -> '%s'", existing.Name, rb.Name))
	}
	if existing.Enabled != rb.Enabled {
		changes = append(changes, fmt.Sprintf("enabled: %v -> %v", existing.Enabled, rb.Enabled))
	}
	if existing.ServiceAccountID != rb.ServiceAccountID {
		changes = append(changes, fmt.Sprintf("SA: %s -> %s", existing.ServiceAccountID, rb.ServiceAccountID))
	}
	changes = append(changes, fmt.Sprintf("roles: %v -> %v", existing.RoleIDs, rb.RoleIDs))

	message := fmt.Sprintf("Updated RoleBinding %s: %s", rb.ID, strings.Join(changes, ", "))

	if err := rbacrepo.SaveRoleBinding(ctx, rb); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRoleBinding", rb.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateRoleBinding", rb.ID, message, true)
	return rb, nil
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
		existing.ID, existing.Name, existing.ServiceAccountID, existing.RoleIDs, existing.Enabled)
	if err := rbacrepo.DeleteRoleBinding(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRoleBinding", id, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteRoleBinding", id, message, true)
	return nil
}
