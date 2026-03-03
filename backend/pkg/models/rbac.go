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
	ID    string       `json:"id"`
	Name  string       `json:"name"`
	Rules []PolicyRule `json:"rules"`
}

func (ro *Role) Bind(r *http.Request) error {
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

type Session struct {
	ID        string `json:"id"`
	UserType  string `json:"userType"`
	CreatedAt string `json:"createdAt"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}

func (s *Session) Bind(r *http.Request) error {
	return nil
}

type SimulatePermissionsRequest struct {
	ServiceAccountID string `json:"serviceAccountId"`
	Verb             string `json:"verb"`
	Resource         string `json:"resource"`
}

func (s *SimulatePermissionsRequest) Bind(r *http.Request) error {
	return nil
}
