package models

type LoginRequest struct {
	Password string `json:"password"`
	Totp     string `json:"totp"`
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

type ServiceAccount struct {
	Name       string `json:"name"`
	Token      string `json:"token"`
	Comments   string `json:"comments"`
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

type RoleBinding struct {
	Name               string   `json:"name"`
	RoleNames          []string `json:"roleNames"`
	ServiceAccountName string   `json:"serviceAccountName"`
	Enabled            bool     `json:"enabled"`
}

type ResourcePermissions struct {
	AllowedAll       bool     `json:"allowedAll"`
	AllowedInstances []string `json:"allowedInstances"`
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
