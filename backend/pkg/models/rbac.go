package models

import "net/http"

type LoginRequest struct {
	Password string `json:"password"`
	Totp     string `json:"totp"`
}

func (l *LoginRequest) Bind(r *http.Request) error {
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
	Name  string       `json:"name"`
	Rules []PolicyRule `json:"rules"`
}

func (ro *Role) Bind(r *http.Request) error {
	return nil
}

type ServiceAccount struct {
	Name       string `json:"name"`
	Token      string `json:"token"`
	Comments   string `json:"comments"`
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

func (s *ServiceAccount) Bind(r *http.Request) error {
	return nil
}

type RoleBinding struct {
	Name               string   `json:"name"`
	RoleNames          []string `json:"roleNames"`
	ServiceAccountName string   `json:"serviceAccountName"`
	Enabled            bool     `json:"enabled"`
}

func (rb *RoleBinding) Bind(r *http.Request) error {
	return nil
}

type ResourcePermissions struct {
	AllowedAll       bool        `json:"allowedAll"`
	AllowedInstances []string    `json:"allowedInstances"`
	MatchedRule      *PolicyRule `json:"matchedRule,omitempty"` // Records which rule allowed the access
}

// IsAllowed precisely checks if a specific resource instance is allowed.
func (p *ResourcePermissions) IsAllowed(resourceName string) bool {
	if p == nil || p.AllowedAll {
		return true
	}
	for _, inst := range p.AllowedInstances {
		if inst == resourceName {
			return true
		}
	}
	return false
}
