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
	authservice "homelab/pkg/services/auth"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

var (
	idRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)
)

// Service Accounts

func ListServiceAccounts(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, errors.New("permission denied: rbac")
	}

	sas, total, err := rbacrepo.ListServiceAccounts(ctx, uint64(page-1), uint(pageSize), search)
	if err != nil {
		return nil, err
	}

	var items []interface{}
	for _, sa := range sas {
		items = append(items, sa)
	}

	return &common.PaginatedResponse{
		Items: items,
		Total: int(total),
		Page:  page,
	}, nil
}

func CreateServiceAccount(ctx context.Context, sa *models.ServiceAccount) (*models.ServiceAccount, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, errors.New("permission denied: rbac")
	}
	if sa.ID == "" {
		return nil, errors.New("id is required")
	}
	if !idRegex.MatchString(sa.ID) {
		return nil, errors.New("id only allows alphanumeric characters, hyphens and underscores")
	}

	existing, _ := rbacrepo.GetServiceAccount(ctx, sa.ID)
	if existing != nil {
		return nil, errors.New("ServiceAccount already exists")
	}

	if sa.Token == "" {
		sa.Token = uuid.New().String()
	}

	sa.Enabled = true

	message := fmt.Sprintf("Created ServiceAccount: %s (id: %s, enabled: %v, comments: '%s')", sa.Name, sa.ID, sa.Enabled, sa.Comments)
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, true)
	return sa, nil
}

func UpdateServiceAccount(ctx context.Context, id string, sa *models.ServiceAccount) (*models.ServiceAccount, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, errors.New("permission denied: rbac")
	}
	if sa.ID != id {
		return nil, errors.New("id in body does not match path")
	}

	existing, err := rbacrepo.GetServiceAccount(ctx, id)
	if err != nil {
		return nil, errors.New("ServiceAccount not found")
	}

	if sa.Token == "" {
		sa.Token = existing.Token
	}

	changes := []string{}
	if existing.Name != sa.Name {
		changes = append(changes, fmt.Sprintf("name: '%s' -> '%s'", existing.Name, sa.Name))
	}
	if existing.Enabled != sa.Enabled {
		changes = append(changes, fmt.Sprintf("enabled: %v -> %v", existing.Enabled, sa.Enabled))
	}
	if existing.Comments != sa.Comments {
		changes = append(changes, fmt.Sprintf("comments: '%s' -> '%s'", existing.Comments, sa.Comments))
	}
	message := fmt.Sprintf("Updated ServiceAccount: %s", sa.ID)
	if len(changes) > 0 {
		message += ": " + strings.Join(changes, ", ")
	} else {
		message += " (no changes)"
	}

	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateServiceAccount", sa.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateServiceAccount", sa.ID, message, true)
	return sa, nil
}

func DeleteServiceAccount(ctx context.Context, id string) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return errors.New("permission denied: rbac")
	}

	existing, err := rbacrepo.GetServiceAccount(ctx, id)
	if err != nil {
		return errors.New("ServiceAccount not found")
	}

	// Cascade delete RoleBindings
	rbs, err := rbacrepo.ListRoleBindingsAll(ctx)
	if err == nil {
		for _, rb := range rbs {
			if rb.ServiceAccountID == id {
				rbacrepo.DeleteRoleBinding(ctx, rb.ID)
			}
		}
	}

	message := fmt.Sprintf("Deleted ServiceAccount: %s (name: %s, enabled: %v, comments: '%s')", existing.ID, existing.Name, existing.Enabled, existing.Comments)
	if err := rbacrepo.DeleteServiceAccount(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteServiceAccount", id, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteServiceAccount", id, message, true)
	return nil
}

func ResetServiceAccountToken(ctx context.Context, id string) (*models.ServiceAccount, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, errors.New("permission denied: rbac")
	}
	sa, err := rbacrepo.GetServiceAccount(ctx, id)
	if err != nil {
		return nil, errors.New("service account not found")
	}

	sa.Token = uuid.New().String()
	message := fmt.Sprintf("Reset token for ServiceAccount: %s", sa.ID)
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", sa.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", sa.ID, message, true)
	return sa, nil
}

// Roles

func ListRoles(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, errors.New("permission denied: rbac")
	}
	roles, total, err := rbacrepo.ListRoles(ctx, uint64(page-1), uint(pageSize), search)
	if err != nil {
		return nil, err
	}

	var items []interface{}
	for _, r := range roles {
		items = append(items, r)
	}

	return &common.PaginatedResponse{
		Items: items,
		Total: int(total),
		Page:  page,
	}, nil
}

func CreateRole(ctx context.Context, role *models.Role) (*models.Role, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, errors.New("permission denied: rbac")
	}
	if role.ID == "" {
		role.ID = uuid.New().String()
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
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, errors.New("permission denied: rbac")
	}
	if role.ID != id {
		return nil, errors.New("id in body does not match path")
	}

	existing, err := rbacrepo.GetRole(ctx, id)
	if err != nil {
		return nil, errors.New("Role not found")
	}

	changes := []string{}
	if existing.Name != role.Name {
		changes = append(changes, fmt.Sprintf("name: '%s' -> '%s'", existing.Name, role.Name))
	}
	// For rules, just log the change if it happened (simplified)
	// Comparing rules can be complex, for now we just log they were updated
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
		return errors.New("permission denied: rbac")
	}

	existing, err := rbacrepo.GetRole(ctx, id)
	if err != nil {
		return errors.New("Role not found")
	}

	// Cascade update/delete RoleBindings
	rbs, err := rbacrepo.ListRoleBindingsAll(ctx)
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

// Role Bindings

func ListRoleBindings(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, errors.New("permission denied: rbac")
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
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, errors.New("permission denied: rbac")
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
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, errors.New("permission denied: rbac")
	}
	if rb.ID != id {
		return nil, errors.New("id in body does not match path")
	}

	existing, err := rbacrepo.GetRoleBinding(ctx, id)
	if err != nil {
		return nil, errors.New("RoleBinding not found")
	}

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
	// Simplified role list comparison
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
		return errors.New("permission denied: rbac")
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

// Simulation

func SimulatePermissions(ctx context.Context, saID, verb, resource string) (*models.ResourcePermissions, error) {
	if saID == "" || verb == "" || resource == "" {
		return nil, errors.New("saID, verb and resource are required")
	}

	return authservice.GetPermissions(ctx, saID, verb, resource)
}
