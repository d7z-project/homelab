package shared

type PaginationRequest struct {
	Cursor string `json:"cursor"`
	Limit  int    `json:"limit"`
	Search string `json:"search"`
}

type PaginationResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor"`
	HasMore    bool   `json:"hasMore"`
}

func (p *PaginationResponse[T]) GetItems() interface{} { return p.Items }
func (p *PaginationResponse[T]) GetCursor() string     { return p.NextCursor }
func (p *PaginationResponse[T]) HasMoreData() bool     { return p.HasMore }
