package discovery

import (
	"context"
	"errors"
	"homelab/pkg/models"
	"strconv"
	"sync"
)

// LookupFunc defines the signature for discovery lookup handlers
type LookupFunc func(ctx context.Context, search string, cursor string, limit int) (*models.PaginationResponse[models.LookupItem], error)

var (
	registry = make(map[string]LookupFunc)
	mu       sync.RWMutex
)

var (
	ErrCodeNotFound = errors.New("lookup code not found")
)

// Register registers a new lookup handler for a given code
func Register(code string, f LookupFunc) {
	mu.Lock()
	defer mu.Unlock()
	registry[code] = f
}

// GetRegisteredCodes returns all registered discovery codes
func GetRegisteredCodes() []string {
	mu.RLock()
	defer mu.RUnlock()
	codes := make([]string, 0, len(registry))
	for code := range registry {
		codes = append(codes, code)
	}
	return codes
}

// Lookup executes a discovery lookup for a given request
func Lookup(ctx context.Context, req models.LookupRequest) (*models.PaginationResponse[models.LookupItem], error) {
	mu.RLock()
	f, ok := registry[req.Code]
	mu.RUnlock()

	if !ok {
		return nil, ErrCodeNotFound
	}

	// For legacy support in LookupRequest, we might still have Offset.
	// We'll treat Offset as a string cursor if Cursor is empty for backward compatibility if needed,
	// but here we prefer the new Cursor field.
	cursor := req.Cursor
	return f(ctx, req.Search, cursor, req.Limit)
}

// Verify checks if a specific ID exists for a given discovery code.
func Verify(ctx context.Context, code string, id string) (bool, error) {
	mu.RLock()
	f, ok := registry[code]
	mu.RUnlock()

	if !ok {
		return false, ErrCodeNotFound
	}

	// Search by ID, fetch first 100
	res, err := f(ctx, id, "", 100)
	if err != nil {
		return false, err
	}

	for _, item := range res.Items {
		if item.ID == id {
			return true, nil
		}
	}

	return false, nil
}

// Paginate applies cursor-based pagination to a slice of LookupItems.
func Paginate(items []models.LookupItem, cursor string, limit int) *models.PaginationResponse[models.LookupItem] {
	total := len(items)
	if limit <= 0 {
		limit = 20
	}

	// For slices, cursor is just the index string
	offset := 0
	if cursor != "" {
		var err error
		offset, err = strconv.Atoi(cursor)
		if err != nil {
			offset = 0
		}
	}

	if offset >= total {
		return &models.PaginationResponse[models.LookupItem]{
			Items:      []models.LookupItem{},
			NextCursor: "",
			HasMore:    false,
			Total:      int64(total),
		}
	}

	end := offset + limit
	hasMore := true
	nextCursor := ""
	if end >= total {
		end = total
		hasMore = false
	} else {
		nextCursor = strconv.Itoa(end)
	}

	return &models.PaginationResponse[models.LookupItem]{
		Items:      items[offset:end],
		NextCursor: nextCursor,
		HasMore:    hasMore,
		Total:      int64(total),
	}
}
