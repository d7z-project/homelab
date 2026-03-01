package rbac

import (
	"context"
	"errors"
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

	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.Name, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.Name, true)
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

	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateServiceAccount", sa.Name, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateServiceAccount", sa.Name, true)
	return sa, nil
}

func DeleteServiceAccount(ctx context.Context, name string) error {
	if err := rbacrepo.DeleteServiceAccount(ctx, name); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteServiceAccount", name, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteServiceAccount", name, true)
	return nil
}

func ResetServiceAccountToken(ctx context.Context, name string) (*models.ServiceAccount, error) {
	sa, err := rbacrepo.GetServiceAccount(ctx, name)
	if err != nil {
		return nil, errors.New("service account not found")
	}

	sa.Token = uuid.New().String()
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", sa.Name, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", sa.Name, true)
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

	if err := rbacrepo.SaveRole(ctx, role); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRole", role.Name, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateRole", role.Name, true)
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

	if err := rbacrepo.SaveRole(ctx, role); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRole", role.Name, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateRole", role.Name, true)
	return role, nil
}

func DeleteRole(ctx context.Context, name string) error {
	if err := rbacrepo.DeleteRole(ctx, name); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRole", name, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteRole", name, true)
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

	if err := rbacrepo.SaveRoleBinding(ctx, rb); err != nil {
		commonaudit.FromContext(ctx).Log("CreateRoleBinding", rb.Name, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateRoleBinding", rb.Name, true)
	return rb, nil
}

func UpdateRoleBinding(ctx context.Context, name string, rb *models.RoleBinding) (*models.RoleBinding, error) {
	if rb.Name != name {
		return nil, errors.New("name in body does not match path")
	}

	_, err := rbacrepo.GetRoleBinding(ctx, name)
	if err != nil {
		return nil, errors.New("RoleBinding not found")
	}

	if err := rbacrepo.SaveRoleBinding(ctx, rb); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateRoleBinding", rb.Name, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateRoleBinding", rb.Name, true)
	return rb, nil
}

func DeleteRoleBinding(ctx context.Context, name string) error {
	if err := rbacrepo.DeleteRoleBinding(ctx, name); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteRoleBinding", name, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteRoleBinding", name, true)
	return nil
}

// Simulation

func SimulatePermissions(ctx context.Context, saName, verb, resource string) (*models.ResourcePermissions, error) {
	if saName == "" || verb == "" || resource == "" {
		return nil, errors.New("saName, verb and resource are required")
	}

	return authservice.GetPermissions(ctx, saName, verb, resource)
}
