package discovery

import (
	"context"
	"errors"
	"homelab/pkg/models"
	"sort"
	"strconv"
	"strings"
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

// ScanCodes returns all registered discovery codes
func ScanCodes() []string {
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

// --- Resource Discovery (RBAC Help) ---

type resourceInfo struct {
	discover DiscoverFunc
	verbs    []string
}

// DiscoverFunc returns a list of matching resource paths based on the remaining prefix
type DiscoverFunc func(ctx context.Context, prefix string) ([]models.DiscoverResult, error)

var (
	discoveredResources = make(map[string]resourceInfo)
	discoveryMu         sync.RWMutex
)

// RegisterResource allows modules to register their resource types
func RegisterResource(name string, f DiscoverFunc) {
	RegisterResourceWithVerbs(name, f, []string{"*"})
}

// RegisterResourceWithVerbs allows modules to register their resource types and supported verbs
func RegisterResourceWithVerbs(name string, f DiscoverFunc, verbs []string) {
	discoveryMu.Lock()
	defer discoveryMu.Unlock()
	discoveredResources[name] = resourceInfo{
		discover: f,
		verbs:    verbs,
	}
}

// SuggestResources returns a list of resource paths matching the prefix
func SuggestResources(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
	discoveryMu.RLock()
	defer discoveryMu.RUnlock()

	suggestions := make([]models.DiscoverResult, 0)
	seen := make(map[string]struct{})
	prefixLower := strings.ToLower(prefix)

	// 1. 匹配根资源 (Root Resources)
	for name := range discoveredResources {
		if strings.HasPrefix(name, prefixLower) {
			if _, exists := seen[name]; !exists {
				suggestions = append(suggestions, models.DiscoverResult{
					FullID: name,
					Name:   name,
					Final:  false,
				})
				seen[name] = struct{}{}
			}
		}
	}

	// 2. 匹配子资源 (Sub-resources)
	for baseRes, info := range discoveredResources {
		if strings.HasPrefix(prefixLower, baseRes+"/") {
			remaining := prefix[len(baseRes)+1:]
			matches, err := info.discover(ctx, remaining)
			if err == nil {
				for _, m := range matches {
					fullID := baseRes + "/" + m.FullID
					if _, exists := seen[fullID]; !exists {
						m.FullID = fullID
						suggestions = append(suggestions, m)
						seen[fullID] = struct{}{}
					}
				}
			}

			// 为当前已确定的根资源添加通配符建议
			wildcards := []string{"*", "**"}
			for _, w := range wildcards {
				if remaining == "" || strings.HasPrefix(w, strings.ToLower(remaining)) {
					fullPath := baseRes + "/" + w
					if _, exists := seen[fullPath]; !exists {
						suggestions = append(suggestions, models.DiscoverResult{
							FullID: fullPath,
							Name:   w,
							Final:  true,
						})
						seen[fullPath] = struct{}{}
					}
				}
			}
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].FullID < suggestions[j].FullID
	})
	return suggestions, nil
}

// SuggestVerbs returns supported verbs for a given resource prefix
func SuggestVerbs(ctx context.Context, resourcePrefix string) ([]string, error) {
	discoveryMu.RLock()
	defer discoveryMu.RUnlock()

	if resourcePrefix == "" {
		return []string{"*"}, nil
	}

	resourcePrefixLower := strings.ToLower(resourcePrefix)

	var bestMatch *resourceInfo
	longestKey := 0

	for name, info := range discoveredResources {
		if resourcePrefixLower == name || strings.HasPrefix(resourcePrefixLower, name+"/") {
			if len(name) > longestKey {
				infoCopy := info
				bestMatch = &infoCopy
				longestKey = len(name)
			}
		}
	}

	if bestMatch != nil {
		return bestMatch.verbs, nil
	}

	return []string{"*"}, nil
}

// --- Service Account Usage Checking ---

var saUsageCheckers []func(ctx context.Context, id string) error

// RegisterSAUsageChecker allows other services to register functions that check if an SA is in use.
func RegisterSAUsageChecker(f func(ctx context.Context, id string) error) {
	saUsageCheckers = append(saUsageCheckers, f)
}

// CheckSAUsage runs all registered checkers to see if a ServiceAccount is being used.
func CheckSAUsage(ctx context.Context, id string) error {
	for _, check := range saUsageCheckers {
		if err := check(ctx, id); err != nil {
			return err
		}
	}
	return nil
}
