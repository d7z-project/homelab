package testkit

import (
	"context"

	rbacmodel "homelab/pkg/models/core/rbac"
	secretmodel "homelab/pkg/models/core/secret"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	authservice "homelab/pkg/services/core/auth"
	secretservice "homelab/pkg/services/core/secret"
)

func SeedServiceAccount(ctx context.Context, id, name string, rules ...rbacmodel.PolicyRule) (string, error) {
	rbacrepo.ClearCache()

	sa := &rbacmodel.ServiceAccount{
		ID: id,
		Meta: rbacmodel.ServiceAccountV1Meta{
			Name:    name,
			Enabled: true,
		},
		Status: rbacmodel.ServiceAccountV1Status{
			HasAuthSecret: true,
		},
		Generation: 1,
	}
	if err := rbacrepo.SaveServiceAccount(ctx, sa); err != nil {
		return "", err
	}

	token, err := authservice.CreateSAToken(id)
	if err != nil {
		return "", err
	}
	if err := secretservice.Put(ctx, secretmodel.OwnerKindServiceAccount, id, secretmodel.PurposeAuthToken, token); err != nil {
		return "", err
	}

	if len(rules) == 0 {
		return token, nil
	}

	roleID := id + "-role"
	if err := rbacrepo.SaveRole(ctx, &rbacmodel.Role{
		ID: roleID,
		Meta: rbacmodel.RoleV1Meta{
			Name:  roleID,
			Rules: append([]rbacmodel.PolicyRule(nil), rules...),
		},
		Generation: 1,
	}); err != nil {
		return "", err
	}

	if err := rbacrepo.SaveRoleBinding(ctx, &rbacmodel.RoleBinding{
		ID: id + "-binding",
		Meta: rbacmodel.RoleBindingV1Meta{
			Name:             id + "-binding",
			ServiceAccountID: id,
			RoleIDs:          []string{roleID},
			Enabled:          true,
		},
		Generation: 1,
	}); err != nil {
		return "", err
	}

	return token, nil
}

type TestingT interface {
	Helper()
	Fatalf(format string, args ...any)
}
