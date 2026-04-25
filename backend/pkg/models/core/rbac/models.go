package rbac

import (
	"context"
	"errors"
	"strings"

	"homelab/pkg/models/shared"
)

type PolicyRule struct {
	Resource string   `json:"resource"`
	Verbs    []string `json:"verbs"`
}

type RoleV1Meta struct {
	Name     string       `json:"name"`
	Comments string       `json:"comments"`
	Rules    []PolicyRule `json:"rules"`
}

func (m *RoleV1Meta) Validate(_ context.Context) error {
	if len(m.Rules) == 0 {
		return errors.New("at least one policy rule is required")
	}
	return nil
}

type RoleV1Status struct{}

type Role = shared.Resource[RoleV1Meta, RoleV1Status]

type ServiceAccountV1Meta struct {
	Name     string `json:"name"`
	Token    string `json:"token"`
	Comments string `json:"comments"`
	Enabled  bool   `json:"enabled"`
}

func (m *ServiceAccountV1Meta) Validate(_ context.Context) error {
	return nil
}

type ServiceAccountV1Status struct {
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

type ServiceAccount = shared.Resource[ServiceAccountV1Meta, ServiceAccountV1Status]

type RoleBindingV1Meta struct {
	Name             string   `json:"name"`
	RoleIDs          []string `json:"roleIds"`
	ServiceAccountID string   `json:"serviceAccountId"`
	Enabled          bool     `json:"enabled"`
}

func (m *RoleBindingV1Meta) Validate(_ context.Context) error {
	if m.ServiceAccountID == "" {
		return errors.New("service account ID is required")
	}
	if len(m.RoleIDs) == 0 {
		return errors.New("at least one role must be assigned")
	}
	return nil
}

type RoleBindingV1Status struct{}

type RoleBinding = shared.Resource[RoleBindingV1Meta, RoleBindingV1Status]

type ResourcePermissions struct {
	AllowedAll       bool        `json:"allowedAll"`
	AllowedInstances []string    `json:"allowedInstances"`
	MatchedRule      *PolicyRule `json:"matchedRule,omitempty"`
}

func (p *ResourcePermissions) IsAllowed(resourceName string) bool {
	if p == nil {
		return false
	}
	if p.AllowedAll {
		return true
	}
	for _, inst := range p.AllowedInstances {
		if inst == resourceName || strings.HasPrefix(resourceName, inst+"/") {
			return true
		}
	}
	return false
}

type SimulatePermissionsRequest struct {
	ServiceAccountID string `json:"serviceAccountId"`
	Verb             string `json:"verb"`
	Resource         string `json:"resource"`
}
