package v1

import (
	"errors"
	"net/http"
	"strings"
)

type PolicyRule struct {
	Resource string   `json:"resource"`
	Verbs    []string `json:"verbs"`
}

type RoleMeta struct {
	Name     string       `json:"name"`
	Comments string       `json:"comments"`
	Rules    []PolicyRule `json:"rules"`
}

type RoleStatus struct{}

type Role struct {
	ID              string     `json:"id"`
	Meta            RoleMeta   `json:"meta"`
	Status          RoleStatus `json:"status"`
	Generation      int64      `json:"generation"`
	ResourceVersion int64      `json:"resourceVersion"`
}

func (r *Role) Bind(_ *http.Request) error {
	r.Meta.Name = strings.TrimSpace(r.Meta.Name)
	r.Meta.Comments = strings.TrimSpace(r.Meta.Comments)
	if len(r.Meta.Rules) == 0 {
		return errors.New("at least one policy rule is required")
	}
	return nil
}

type ServiceAccountMeta struct {
	Name     string `json:"name"`
	Token    string `json:"token"`
	Comments string `json:"comments"`
	Enabled  bool   `json:"enabled"`
}

type ServiceAccountStatus struct {
	LastUsedAt string `json:"lastUsedAt,omitempty"`
}

type ServiceAccount struct {
	ID              string               `json:"id"`
	Meta            ServiceAccountMeta   `json:"meta"`
	Status          ServiceAccountStatus `json:"status"`
	Generation      int64                `json:"generation"`
	ResourceVersion int64                `json:"resourceVersion"`
}

func (s *ServiceAccount) Bind(_ *http.Request) error {
	s.Meta.Name = strings.TrimSpace(s.Meta.Name)
	s.Meta.Comments = strings.TrimSpace(s.Meta.Comments)
	return nil
}

type RoleBindingMeta struct {
	Name             string   `json:"name"`
	RoleIDs          []string `json:"roleIds"`
	ServiceAccountID string   `json:"serviceAccountId"`
	Enabled          bool     `json:"enabled"`
}

type RoleBindingStatus struct{}

type RoleBinding struct {
	ID              string            `json:"id"`
	Meta            RoleBindingMeta   `json:"meta"`
	Status          RoleBindingStatus `json:"status"`
	Generation      int64             `json:"generation"`
	ResourceVersion int64             `json:"resourceVersion"`
}

func (r *RoleBinding) Bind(_ *http.Request) error {
	r.Meta.Name = strings.TrimSpace(r.Meta.Name)
	if r.Meta.Name == "" {
		return errors.New("role binding name is required")
	}
	if r.Meta.ServiceAccountID == "" {
		return errors.New("service account ID is required")
	}
	if len(r.Meta.RoleIDs) == 0 {
		return errors.New("at least one role must be assigned")
	}
	return nil
}

type ResourcePermissions struct {
	AllowedAll       bool        `json:"allowedAll"`
	AllowedInstances []string    `json:"allowedInstances"`
	MatchedRule      *PolicyRule `json:"matchedRule,omitempty"`
}

type SimulatePermissionsRequest struct {
	ServiceAccountID string `json:"serviceAccountId"`
	Verb             string `json:"verb"`
	Resource         string `json:"resource"`
}

func (r *SimulatePermissionsRequest) Bind(_ *http.Request) error {
	if r.ServiceAccountID == "" {
		return errors.New("service account ID is required")
	}
	if r.Verb == "" {
		return errors.New("verb is required")
	}
	if r.Resource == "" {
		return errors.New("resource is required")
	}
	return nil
}
