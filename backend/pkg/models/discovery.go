package models

import (
	"net/http"
)

// LookupItem represents a single item in a dropdown or selection list
type LookupItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Icon        string `json:"icon,omitempty"` // Optional icon name for M3
}

// LookupRequest defines the query parameters for discovery lookups
type LookupRequest struct {
	Code   string `json:"code"`
	Search string `json:"search"`
	Offset int    `json:"offset"`
	Limit  int    `json:"limit"`
}

func (l *LookupRequest) Bind(r *http.Request) error {
	if l.Limit <= 0 {
		l.Limit = 20
	}
	if l.Limit > 100 {
		l.Limit = 100
	}
	return nil
}

// LookupResponse contains the list of items and total count for pagination
type LookupResponse struct {
	Items []LookupItem `json:"items"`
	Total int          `json:"total"`
}

func (l *LookupResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
