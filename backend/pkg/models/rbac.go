package models

import (
	"errors"
	"fmt"
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

type Role struct {
	ID       string       `json:"id"`
	Name     string       `json:"name"`
	Comments string       `json:"comments"`
	Rules    []PolicyRule `json:"rules"`
}

func (ro *Role) Bind(r *http.Request) error {
	ro.ID = strings.TrimSpace(ro.ID)
	if ro.ID == "" {
		return nil // ID is optional on create (UUID generated)
	}
	if !rbacIdRegex.MatchString(ro.ID) {
		return fmt.Errorf("invalid role ID format: %s", ro.ID)
	}
	if len(ro.Rules) == 0 {
		return errors.New("at least one policy rule is required")
	}
	return nil
}

type ServiceAccount struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Token      string `json:"token"`
	Comments   string `json:"comments"`
	Enabled    bool   `json:"enabled"`
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

func (s *ServiceAccount) Bind(r *http.Request) error {
	s.ID = strings.TrimSpace(s.ID)
	if s.ID == "" {
		return errors.New("service account ID is required")
	}
	if !rbacIdRegex.MatchString(s.ID) {
		return fmt.Errorf("invalid service account ID format: %s", s.ID)
	}
	return nil
}

type RoleBinding struct {
	ID               string   `json:"id"`
	Name             string   `json:"name"`
	RoleIDs          []string `json:"roleIds"`
	ServiceAccountID string   `json:"serviceAccountId"`
	Enabled          bool     `json:"enabled"`
}

func (rb *RoleBinding) Bind(r *http.Request) error {
	rb.Name = strings.TrimSpace(rb.Name)
	if rb.Name == "" {
		return errors.New("role binding name is required")
	}
	if rb.ServiceAccountID == "" {
		return errors.New("service account ID is required")
	}
	if len(rb.RoleIDs) == 0 {
		return errors.New("at least one role must be assigned")
	}
	return nil
}

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
