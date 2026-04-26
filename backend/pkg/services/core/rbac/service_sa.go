package rbac

import (
	"context"
	"errors"
	"fmt"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	secretmodel "homelab/pkg/models/core/secret"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	authservice "homelab/pkg/services/core/auth"
	discoveryservice "homelab/pkg/services/core/discovery"
	secretservice "homelab/pkg/services/core/secret"

	rbacmodel "homelab/pkg/models/core/rbac"
)

func CreateServiceAccount(ctx context.Context, sa *rbacmodel.ServiceAccount) (*rbacmodel.ServiceAccount, string, error) {
	if sa.ID == "" {
		return nil, "", errors.New("ServiceAccount ID is required")
	}
	if err := normalizeServiceAccount(sa); err != nil {
		return nil, "", err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, "", fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	existing, _ := rbacrepo.GetServiceAccount(ctx, sa.ID)
	if existing != nil {
		return nil, "", errors.New("ServiceAccount already exists")
	}

	plainToken, err := authservice.CreateSAToken(sa.ID)
	if err != nil {
		return nil, "", err
	}
	sa.Meta.Enabled = true
	sa.Status.HasAuthSecret = false

	err = rbacrepo.SaveServiceAccount(ctx, sa)

	message := fmt.Sprintf("Created ServiceAccount: %s (id: %s, enabled: %v)", sa.Meta.Name, sa.ID, sa.Meta.Enabled)
	if err != nil {
		commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, false)
		return nil, "", err
	}

	if err := secretservice.Put(ctx, secretmodel.OwnerKindServiceAccount, sa.ID, secretmodel.PurposeAuthToken, plainToken); err != nil {
		_ = rbacrepo.DeleteServiceAccount(ctx, sa.ID)
		commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, false)
		return nil, "", err
	}
	if err := rbacrepo.UpdateServiceAccountStatus(ctx, sa.ID, func(status *rbacmodel.ServiceAccountV1Status) {
		status.HasAuthSecret = true
	}); err != nil {
		_ = secretservice.Delete(ctx, secretmodel.OwnerKindServiceAccount, sa.ID, secretmodel.PurposeAuthToken)
		_ = rbacrepo.DeleteServiceAccount(ctx, sa.ID)
		commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, false)
		return nil, "", err
	}

	commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, true)

	updated, _ := rbacrepo.GetServiceAccount(ctx, sa.ID)
	return updated, plainToken, nil
}

func UpdateServiceAccount(ctx context.Context, id string, sa *rbacmodel.ServiceAccount) (*rbacmodel.ServiceAccount, error) {
	if err := normalizeServiceAccount(sa); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	existing, err := rbacrepo.GetServiceAccount(ctx, id)
	if err != nil {
		return nil, err
	}
	existing.Meta.Name = sa.Meta.Name
	existing.Meta.Enabled = sa.Meta.Enabled
	existing.Meta.Comments = sa.Meta.Comments
	err = rbacrepo.SaveServiceAccount(ctx, existing)

	message := fmt.Sprintf("Updated ServiceAccount %s", id)
	if err != nil {
		commonaudit.FromContext(ctx).Log("UpdateServiceAccount", id, message, false)
		return nil, err
	}

	updated, _ := rbacrepo.GetServiceAccount(ctx, id)
	commonaudit.FromContext(ctx).Log("UpdateServiceAccount", id, message, true)
	return updated, nil
}

func DeleteServiceAccount(ctx context.Context, discovery *discoveryservice.Service, id string) error {
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

	if discovery == nil {
		return fmt.Errorf("discovery service not configured")
	}
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
	if existing.Status.HasAuthSecret {
		if err := secretservice.Delete(ctx, secretmodel.OwnerKindServiceAccount, id, secretmodel.PurposeAuthToken); err != nil {
			commonaudit.FromContext(ctx).Log("DeleteServiceAccount", id, message, false)
			return err
		}
	}
	if err := rbacrepo.DeleteServiceAccount(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteServiceAccount", id, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteServiceAccount", id, message, true)
	return nil
}

func ResetServiceAccountToken(ctx context.Context, id string) (*rbacmodel.ServiceAccount, string, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, "", fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	updated, err := rbacrepo.GetServiceAccount(ctx, id)
	if err != nil {
		return nil, "", err
	}
	plainToken, err := authservice.CreateSAToken(id)
	if err != nil {
		return nil, "", err
	}
	err = secretservice.Put(ctx, secretmodel.OwnerKindServiceAccount, id, secretmodel.PurposeAuthToken, plainToken)

	message := fmt.Sprintf("Reset token for ServiceAccount: %s", id)
	if err != nil {
		commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", id, message, false)
		return nil, "", err
	}
	if err := rbacrepo.UpdateServiceAccountStatus(ctx, id, func(status *rbacmodel.ServiceAccountV1Status) {
		status.HasAuthSecret = true
	}); err != nil {
		commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", id, message, false)
		return nil, "", err
	}

	updated, _ = rbacrepo.GetServiceAccount(ctx, id)
	commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", id, message, true)

	return updated, plainToken, nil
}
