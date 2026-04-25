package rbac

import (
	"context"
	"errors"
	"strings"

	rbacmodel "homelab/pkg/models/core/rbac"
)

func normalizeServiceAccount(sa *rbacmodel.ServiceAccount) error {
	if sa == nil {
		return errors.New("service account is required")
	}
	sa.Meta.Name = strings.TrimSpace(sa.Meta.Name)
	sa.Meta.Comments = strings.TrimSpace(sa.Meta.Comments)
	return nil
}

func normalizeRole(role *rbacmodel.Role) error {
	if role == nil {
		return errors.New("role is required")
	}
	role.Meta.Name = strings.TrimSpace(role.Meta.Name)
	role.Meta.Comments = strings.TrimSpace(role.Meta.Comments)
	return role.Meta.Validate(context.Background())
}

func normalizeRoleBinding(rb *rbacmodel.RoleBinding) error {
	if rb == nil {
		return errors.New("role binding is required")
	}
	rb.Meta.Name = strings.TrimSpace(rb.Meta.Name)
	if rb.Meta.Name == "" {
		return errors.New("role binding name is required")
	}
	return rb.Meta.Validate(context.Background())
}
