package rbac

import (
	"context"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	"homelab/pkg/models"
	rbacrepo "homelab/pkg/repositories/rbac"
	authservice "homelab/pkg/services/auth"

	"github.com/google/uuid"
)

// Service Accounts

func ListServiceAccounts(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
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
	if sa.Name == "" {
		return nil, errors.New("name is required")
	}

	existing, _ := rbacrepo.GetServiceAccount(ctx, sa.Name)
	if existing != nil {
		return nil, errors.New("ServiceAccount already exists")
	}

	if sa.Token == "" {
		sa.Token = uuid.New().String()
	}

	message := fmt.Sprintf("Created ServiceAccount: %s", sa.Name)
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.Name, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.Name, message, true)
	return sa, nil
}

func UpdateServiceAccount(ctx context.Context, name string, sa *models.ServiceAccount) (*models.ServiceAccount, error) {
	if sa.Name != name {
		return nil, errors.New("name in body does not match path")
	}

	existing, err := rbacrepo.GetServiceAccount(ctx, name)
	if err != nil {
		return nil, errors.New("ServiceAccount not found")
	}

	if sa.Token == "" {
		sa.Token = existing.Token
	}

	message := fmt.Sprintf("Updated ServiceAccount: %s", sa.Name)
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateServiceAccount", sa.Name, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateServiceAccount", sa.Name, message, true)
	return sa, nil
}

func DeleteServiceAccount(ctx context.Context, name string) error {
	// Cascade delete RoleBindings
	rbs, err := rbacrepo.ListRoleBindingsAll(ctx)
	if err == nil {
		for _, rb := range rbs {
			if rb.ServiceAccountName == name {
				rbacrepo.DeleteRoleBinding(ctx, rb.Name)
			}
		}
	}

	message := fmt.Sprintf("Deleted ServiceAccount: %s", name)
	if err := rbacrepo.DeleteServiceAccount(ctx, name); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteServiceAccount", name, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteServiceAccount", name, message, true)
	return nil
}

func ResetServiceAccountToken(ctx context.Context, name string) (*models.ServiceAccount, error) {
	sa, err := rbacrepo.GetServiceAccount(ctx, name)
	if err != nil {
		return nil, errors.New("service account not found")
	}

	sa.Token = uuid.New().String()
	message := fmt.Sprintf("Reset token for ServiceAccount: %s", sa.Name)
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", sa.Name, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", sa.Name, message, true)
	return sa, nil
}

// Roles

func ListRoles(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
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
	if role.Name == "" {
		return nil, errors.New("name is required")
	}

	existing, _ := rbacrepo.GetRole(ctx, role.Name)
	if existing != nil {
		return nil, errors.New("Role already exists")
	}

	message := fmt.Sprintf("Created Role: %s with %d rules", role.Name, len(role.Rules))
	if err := rbacrepo.SaveRole(ctx, role); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRole", role.Name, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateRole", role.Name, message, true)
	return role, nil
}

func UpdateRole(ctx context.Context, name string, role *models.Role) (*models.Role, error) {
	if role.Name != name {
		return nil, errors.New("name in body does not match path")
	}

	_, err := rbacrepo.GetRole(ctx, name)
	if err != nil {
		return nil, errors.New("Role not found")
	}

	message := fmt.Sprintf("Updated Role: %s", role.Name)
	if err := rbacrepo.SaveRole(ctx, role); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRole", role.Name, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateRole", role.Name, message, true)
	return role, nil
}

func DeleteRole(ctx context.Context, name string) error {
	// Cascade update/delete RoleBindings
	rbs, err := rbacrepo.ListRoleBindingsAll(ctx)
	if err == nil {
		for _, rb := range rbs {
			newRoles := make([]string, 0)
			found := false
			for _, r := range rb.RoleNames {
				if r == name {
					found = true
				} else {
					newRoles = append(newRoles, r)
				}
			}
			if found {
				if len(newRoles) == 0 {
					rbacrepo.DeleteRoleBinding(ctx, rb.Name)
				} else {
					rb.RoleNames = newRoles
					rbacrepo.SaveRoleBinding(ctx, &rb)
				}
			}
		}
	}

	message := fmt.Sprintf("Deleted Role: %s", name)
	if err := rbacrepo.DeleteRole(ctx, name); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRole", name, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteRole", name, message, true)
	return nil
}

// Role Bindings

func ListRoleBindings(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
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
	if rb.Name == "" {
		return nil, errors.New("name is required")
	}

	existing, _ := rbacrepo.GetRoleBinding(ctx, rb.Name)
	if existing != nil {
		return nil, errors.New("RoleBinding already exists")
	}

	message := fmt.Sprintf("Created RoleBinding: %s (SA: %s -> Roles: %v)",
		rb.Name, rb.ServiceAccountName, rb.RoleNames)
	if err := rbacrepo.SaveRoleBinding(ctx, rb); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRoleBinding", rb.Name, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateRoleBinding", rb.Name, message, true)
	return rb, nil
}

func UpdateRoleBinding(ctx context.Context, name string, rb *models.RoleBinding) (*models.RoleBinding, error) {
	if rb.Name != name {
		return nil, errors.New("name in body does not match path")
	}

	existing, err := rbacrepo.GetRoleBinding(ctx, name)
	if err != nil {
		return nil, errors.New("RoleBinding not found")
	}

	message := fmt.Sprintf("Updated RoleBinding: %s", rb.Name)
	if existing.Enabled != rb.Enabled {
		message += fmt.Sprintf(" (Status: %v -> %v)", existing.Enabled, rb.Enabled)
	}

	if err := rbacrepo.SaveRoleBinding(ctx, rb); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRoleBinding", rb.Name, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateRoleBinding", rb.Name, message, true)
	return rb, nil
}

func DeleteRoleBinding(ctx context.Context, name string) error {
	message := fmt.Sprintf("Deleted RoleBinding: %s", name)
	if err := rbacrepo.DeleteRoleBinding(ctx, name); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRoleBinding", name, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteRoleBinding", name, message, true)
	return nil
}

// Simulation

func SimulatePermissions(ctx context.Context, saName, verb, resource string) (*models.ResourcePermissions, error) {
	if saName == "" || verb == "" || resource == "" {
		return nil, errors.New("saName, verb and resource are required")
	}

	return authservice.GetPermissions(ctx, saName, verb, resource)
}
