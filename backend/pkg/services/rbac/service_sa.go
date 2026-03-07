package rbac

import (
	"context"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	actionsrepo "homelab/pkg/repositories/actions"
	rbacrepo "homelab/pkg/repositories/rbac"
	authservice "homelab/pkg/services/auth"
	"strings"
)

func ListServiceAccounts(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
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
	if err := sa.Bind(nil); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	existing, _ := rbacrepo.GetServiceAccount(ctx, sa.ID)
	if existing != nil {
		return nil, errors.New("ServiceAccount already exists")
	}

	plainToken := sa.Token
	if sa.Token == "" {
		token, err := authservice.CreateSAToken(sa.ID)
		if err != nil {
			return nil, err
		}
		plainToken = token
	}

	// Always store hash
	sa.Token = authservice.HashToken(plainToken)
	sa.Enabled = true

	message := fmt.Sprintf("Created ServiceAccount: %s (id: %s, enabled: %v)", sa.Name, sa.ID, sa.Enabled)
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, true)

	// Set back plain token for the response
	sa.Token = plainToken
	return sa, nil
}

func UpdateServiceAccount(ctx context.Context, id string, sa *models.ServiceAccount) (*models.ServiceAccount, error) {
	if err := sa.Bind(nil); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
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

	sa.ID = id
	changes := []string{}
	if existing.Name != sa.Name {
		changes = append(changes, fmt.Sprintf("name: '%s' -> '%s'", existing.Name, sa.Name))
	}
	if existing.Enabled != sa.Enabled {
		changes = append(changes, fmt.Sprintf("enabled: %v -> %v", existing.Enabled, sa.Enabled))
	}
	if existing.Comments != sa.Comments {
		changes = append(changes, "comments updated")
	}

	message := fmt.Sprintf("Updated ServiceAccount %s: %s", sa.ID, strings.Join(changes, ", "))
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateServiceAccount", sa.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateServiceAccount", sa.ID, message, true)
	return sa, nil
}

func DeleteServiceAccount(ctx context.Context, id string) error {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	release, err := lockRBAC(ctx, "global")
	if err != nil {
		return err
	}
	defer release()

	existing, err := rbacrepo.GetServiceAccount(ctx, id)
	if err != nil {
		return errors.New("ServiceAccount not found")
	}

	// Check if used by any Workflow (use global list workflows from repo to avoid circular dependency)
	workflows, err := actionsrepo.ListWorkflows(ctx)
	if err == nil {
		for _, wf := range workflows {
			if wf.ServiceAccountID == id {
				return fmt.Errorf("cannot delete ServiceAccount: it is being used by workflow '%s' (%s)", wf.Name, wf.ID)
			}
		}
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
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}
	sa, err := rbacrepo.GetServiceAccount(ctx, id)
	if err != nil {
		return nil, errors.New("service account not found")
	}

	token, err := authservice.CreateSAToken(sa.ID)
	if err != nil {
		return nil, err
	}

	plainToken := token
	sa.Token = authservice.HashToken(plainToken)

	message := fmt.Sprintf("Reset token for ServiceAccount: %s", sa.ID)
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", sa.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", sa.ID, message, true)

	sa.Token = plainToken
	return sa, nil
}
