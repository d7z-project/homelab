package rbac

import (
	"context"
	"errors"
	"fmt"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	rbacrepo "homelab/pkg/repositories/rbac"
	authservice "homelab/pkg/services/auth"
	"homelab/pkg/services/discovery"
	"strings"
)

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

	plainToken := sa.Meta.Token
	if sa.Meta.Token == "" {
		token, err := authservice.CreateSAToken(sa.ID)
		if err != nil {
			return nil, err
		}
		plainToken = token
	}

	// Always store hash
	sa.Meta.Token = authservice.HashToken(plainToken)
	sa.Meta.Enabled = true

	message := fmt.Sprintf("Created ServiceAccount: %s (id: %s, enabled: %v)", sa.Meta.Name, sa.ID, sa.Meta.Enabled)
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, true)

	// Set back plain token for the response
	sa.Meta.Token = plainToken
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

	if sa.Meta.Token == "" {
		sa.Meta.Token = existing.Meta.Token
	}

	sa.ID = id
	changes := []string{}
	if existing.Meta.Name != sa.Meta.Name {
		changes = append(changes, fmt.Sprintf("name: '%s' -> '%s'", existing.Meta.Name, sa.Meta.Name))
	}
	if existing.Meta.Enabled != sa.Meta.Enabled {
		changes = append(changes, fmt.Sprintf("enabled: %v -> %v", existing.Meta.Enabled, sa.Meta.Enabled))
	}
	if existing.Meta.Comments != sa.Meta.Comments {
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

	// Usage Check
	if err := discovery.CheckSAUsage(ctx, id); err != nil {
		return err
	}

	// Cascade delete RoleBindings
	rbs, err := rbacrepo.ScanAllRoleBindings(ctx)
	if err == nil {
		for _, rb := range rbs {
			if rb.Meta.ServiceAccountID == id {
				_ = rbacrepo.DeleteRoleBinding(ctx, rb.ID)
			}
		}
	}

	message := fmt.Sprintf("Deleted ServiceAccount: %s (name: %s, enabled: %v, comments: '%s')", existing.ID, existing.Meta.Name, existing.Meta.Enabled, existing.Meta.Comments)
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
	sa.Meta.Token = authservice.HashToken(plainToken)

	message := fmt.Sprintf("Reset token for ServiceAccount: %s", sa.ID)
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", sa.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", sa.ID, message, true)

	sa.Meta.Token = plainToken
	return sa, nil
}
