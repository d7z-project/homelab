package models

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"
)

var rbacIdRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

type LoginRequest struct {
	Password string `json:"password"`
	Totp     string `json:"totp"`
}

func (l *LoginRequest) Bind(r *http.Request) error {
	if l.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

type LoginResponse struct {
	SessionID string `json:"session_id"`
}

type PolicyRule struct {
	Resource string   `json:"resource"`
	Verbs    []string `json:"verbs"`
}

type RoleV1Meta struct {
	Name     string       `json:"name"`
	Comments string       `json:"comments"`
	Rules    []PolicyRule `json:"rules"`
}

func (m *RoleV1Meta) Validate(ctx context.Context) error {
	if len(m.Rules) == 0 {
		return errors.New("at least one policy rule is required")
	}
	return nil
}

func (m *RoleV1Meta) Bind(r *http.Request) error {
	return nil
}

type RoleV1Status struct {
}

type Role = Resource[RoleV1Meta, RoleV1Status]

type ServiceAccountV1Meta struct {
	Name     string `json:"name"`
	Token    string `json:"token"`
	Comments string `json:"comments"`
	Enabled  bool   `json:"enabled"`
}

func (m *ServiceAccountV1Meta) Validate(ctx context.Context) error {
	return nil
}

func (m *ServiceAccountV1Meta) Bind(r *http.Request) error {
	return nil
}

type ServiceAccountV1Status struct {
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

type ServiceAccount = Resource[ServiceAccountV1Meta, ServiceAccountV1Status]

type RoleBindingV1Meta struct {
	Name             string   `json:"name"`
	RoleIDs          []string `json:"roleIds"`
	ServiceAccountID string   `json:"serviceAccountId"`
	Enabled          bool     `json:"enabled"`
}

func (m *RoleBindingV1Meta) Validate(ctx context.Context) error {
	if m.ServiceAccountID == "" {
		return errors.New("service account ID is required")
	}
	if len(m.RoleIDs) == 0 {
		return errors.New("at least one role must be assigned")
	}
	return nil
}

func (m *RoleBindingV1Meta) Bind(r *http.Request) error {
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return errors.New("role binding name is required")
	}
	return nil
}

type RoleBindingV1Status struct {
}

type RoleBinding = Resource[RoleBindingV1Meta, RoleBindingV1Status]

type ResourcePermissions struct {
	AllowedAll       bool        `json:"allowedAll"`
	AllowedInstances []string    `json:"allowedInstances"`
	MatchedRule      *PolicyRule `json:"matchedRule,omitempty"` // Records which rule allowed the access
}

// IsAllowed precisely checks if a specific resource instance or prefix is allowed.
// For example, if resourceName is "actions/wf-1", it matches if AllowedAll is true,
// OR if "actions/wf-1" is in AllowedInstances, OR if "actions" is in AllowedInstances.
func (p *ResourcePermissions) IsAllowed(resourceName string) bool {
	if p == nil {
		return false
	}
	if p.AllowedAll {
		return true
	}
	for _, inst := range p.AllowedInstances {
		// Exact match or prefix match (e.g., "actions" matches "actions/wf-1")
		if inst == resourceName || strings.HasPrefix(resourceName, inst+"/") {
			return true
		}
	}
	return false
}

type Session struct {
	ID        string `json:"id"`
	UserType  string `json:"userType"`
	CreatedAt string `json:"createdAt"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}

func (s *Session) Bind(r *http.Request) error {
	if s.ID == "" {
		return errors.New("session ID is required")
	}
	return nil
}

type SimulatePermissionsRequest struct {
	ServiceAccountID string `json:"serviceAccountId"`
	Verb             string `json:"verb"`
	Resource         string `json:"resource"`
}

type DiscoverResult struct {
	FullID string `json:"fullId"` // Actual resource path used in RBAC check
	Name   string `json:"name"`   // Display name in UI
	Final  bool   `json:"final"`  // True if this is a complete resource, False if it's a category/prefix
}

func (s *SimulatePermissionsRequest) Bind(r *http.Request) error {
	if s.ServiceAccountID == "" {
		return errors.New("service account ID is required")
	}
	if s.Verb == "" {
		return errors.New("verb is required")
	}
	if s.Resource == "" {
		return errors.New("resource is required")
	}
	return nil
}
