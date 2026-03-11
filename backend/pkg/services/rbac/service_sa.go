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
)

func CreateServiceAccount(ctx context.Context, sa *models.ServiceAccount) (*models.ServiceAccount, error) {
	if sa.ID == "" {
		return nil, errors.New("ServiceAccount ID is required")
	}
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

	err := rbacrepo.ServiceAccountRepo.Cow(ctx, sa.ID, func(res *models.Resource[models.ServiceAccountV1Meta, models.ServiceAccountV1Status]) error {
		res.Meta = sa.Meta
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	})

	message := fmt.Sprintf("Created ServiceAccount: %s (id: %s, enabled: %v)", sa.Meta.Name, sa.ID, sa.Meta.Enabled)
	if err != nil {
		commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateServiceAccount", sa.ID, message, true)

	// Set back plain token for the response
	updated, _ := rbacrepo.GetServiceAccount(ctx, sa.ID)
	updated.Meta.Token = plainToken
	return updated, nil
}

func UpdateServiceAccount(ctx context.Context, id string, sa *models.ServiceAccount) (*models.ServiceAccount, error) {
	if err := sa.Bind(nil); err != nil {
		return nil, err
	}
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("rbac") {
		return nil, fmt.Errorf("%w: rbac", commonauth.ErrPermissionDenied)
	}

	err := rbacrepo.ServiceAccountRepo.PatchMeta(ctx, id, sa.Generation, func(m *models.ServiceAccountV1Meta) {
		m.Name = sa.Meta.Name
		m.Enabled = sa.Meta.Enabled
		m.Comments = sa.Meta.Comments
	})

	message := fmt.Sprintf("Updated ServiceAccount %s", id)
	if err != nil {
		commonaudit.FromContext(ctx).Log("UpdateServiceAccount", id, message, false)
		return nil, err
	}

	updated, _ := rbacrepo.GetServiceAccount(ctx, id)
	commonaudit.FromContext(ctx).Log("UpdateServiceAccount", id, message, true)
	return updated, nil
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

	var plainToken string
	err := rbacrepo.ServiceAccountRepo.PatchMeta(ctx, id, 0, func(m *models.ServiceAccountV1Meta) {
		token, _ := authservice.CreateSAToken(id)
		plainToken = token
		m.Token = authservice.HashToken(plainToken)
	})

	message := fmt.Sprintf("Reset token for ServiceAccount: %s", id)
	if err != nil {
		commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", id, message, false)
		return nil, err
	}

	updated, _ := rbacrepo.GetServiceAccount(ctx, id)
	commonaudit.FromContext(ctx).Log("ResetServiceAccountToken", id, message, true)

	updated.Meta.Token = plainToken
	return updated, nil
}
