package discovery

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

type DiscoverResult struct {
	FullID string `json:"fullId"`
	Name   string `json:"name"`
	Final  bool   `json:"final"`
}
