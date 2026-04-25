package v1

import (
	"errors"
	"net/http"
)

type LookupItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"`
}

type LookupRequest struct {
	Code   string `json:"code"`
	Search string `json:"search"`
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
}

func (r *LookupRequest) Bind(_ *http.Request) error {
	if r.Code == "" {
		return errors.New("code is required")
	}
	if r.Limit <= 0 {
		r.Limit = 20
	}
	if r.Limit > 100 {
		r.Limit = 100
	}
	return nil
}

type DiscoverResult struct {
	FullID string `json:"fullId"`
	Name   string `json:"name"`
	Final  bool   `json:"final"`
}
