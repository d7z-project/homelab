package models

import (
	"net/http"
)

// PaginationRequest defines the universal cursor-based pagination request
type PaginationRequest struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
}

func (p *PaginationRequest) Bind(r *http.Request) error {
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	return nil
}

// PaginationResponse is a generic container for cursor-based results
type PaginationResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor"`
	HasMore    bool   `json:"hasMore"`
	Total      int64  `json:"total"` // Optional: total count if available
}

func (p *PaginationResponse[T]) GetItems() interface{} { return p.Items }
func (p *PaginationResponse[T]) GetTotal() int64       { return p.Total }
func (p *PaginationResponse[T]) GetCursor() string     { return p.NextCursor }
func (p *PaginationResponse[T]) HasMoreData() bool     { return p.HasMore }

func (p *PaginationResponse[T]) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
