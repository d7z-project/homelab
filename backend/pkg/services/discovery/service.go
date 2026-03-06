package discovery

import (
	"context"
	"errors"
	"homelab/pkg/models"
	"sync"
)

// LookupFunc defines the signature for discovery lookup handlers
type LookupFunc func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error)

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

// Lookup executes a discovery lookup for a given request
func Lookup(ctx context.Context, req models.LookupRequest) ([]models.LookupItem, int, error) {
	mu.RLock()
	f, ok := registry[req.Code]
	mu.RUnlock()

	if !ok {
		return nil, 0, ErrCodeNotFound
	}

	return f(ctx, req.Search, req.Offset, req.Limit)
}

// Verify checks if a specific ID exists for a given discovery code.
// It uses the registered LookupFunc with the ID as search term and checks for an exact match.
func Verify(ctx context.Context, code string, id string) (bool, error) {
	mu.RLock()
	f, ok := registry[code]
	mu.RUnlock()

	if !ok {
		return false, ErrCodeNotFound
	}

	// We search for the ID. We assume the search term will match the ID.
	// Since we need an exact match, we fetch a reasonable amount and check locally.
	items, _, err := f(ctx, id, 0, 100)
	if err != nil {
		return false, err
	}

	for _, item := range items {
		if item.ID == id {
			return true, nil
		}
	}

	return false, nil
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
