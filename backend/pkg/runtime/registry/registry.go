package registry

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"

	metav1 "homelab/pkg/apis/meta/v1"
	discoverymodel "homelab/pkg/models/core/discovery"
	"homelab/pkg/models/shared"
)

type LookupFunc func(ctx context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error)

type ResourceDescriptor struct {
	Group        string
	Resource     string
	Kind         string
	Verbs        []string
	Scope        string
	DiscoverFunc func(ctx context.Context, prefix string, cursor string, limit int) (*metav1.List[discoverymodel.LookupItem], error)
}

type ActionParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type ActionDescriptor struct {
	ID           string
	Category     string
	Title        string
	Description  string
	Params       []ActionParam
	LookupScopes map[string]string
	Permissions  []string
}

type Registry struct {
	mu              sync.RWMutex
	resources       map[string]ResourceDescriptor
	actions         map[string]ActionDescriptor
	lookups         map[string]LookupFunc
	saUsageCheckers []func(ctx context.Context, id string) error
}

func New() *Registry {
	return &Registry{
		resources: make(map[string]ResourceDescriptor),
		actions:   make(map[string]ActionDescriptor),
		lookups:   make(map[string]LookupFunc),
	}
}

var ErrCodeNotFound = errors.New("lookup code not found")

func (r *Registry) RegisterResource(desc ResourceDescriptor) error {
	key := resourceKey(desc.Group, desc.Resource)
	if key == "" {
		return errors.New("resource group and resource are required")
	}
	if desc.Kind == "" {
		return errors.New("resource kind is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.resources[key]; exists {
		return fmt.Errorf("resource %s already registered", key)
	}
	desc.Verbs = normalizeStrings(desc.Verbs)
	r.resources[key] = desc
	return nil
}

func (r *Registry) RegisterAction(desc ActionDescriptor) error {
	if strings.TrimSpace(desc.ID) == "" {
		return errors.New("action id is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.actions[desc.ID]; exists {
		return fmt.Errorf("action %s already registered", desc.ID)
	}
	r.actions[desc.ID] = desc
	return nil
}

func (r *Registry) RegisterLookup(code string, fn LookupFunc) error {
	code = strings.Trim(strings.ToLower(code), "/")
	if code == "" {
		return errors.New("lookup code is required")
	}
	if fn == nil {
		return errors.New("lookup function is required")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.lookups[code]; exists {
		return fmt.Errorf("lookup %s already registered", code)
	}
	r.lookups[code] = fn
	return nil
}

func (r *Registry) ScanCodes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]string, 0, len(r.lookups))
	for code := range r.lookups {
		items = append(items, code)
	}
	sort.Strings(items)
	return items
}

func (r *Registry) Lookup(ctx context.Context, req discoverymodel.LookupRequest) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
	r.mu.RLock()
	fn, ok := r.lookups[strings.Trim(strings.ToLower(req.Code), "/")]
	r.mu.RUnlock()
	if !ok {
		return nil, ErrCodeNotFound
	}
	return fn(ctx, req.Search, req.Cursor, req.Limit)
}

func (r *Registry) Verify(ctx context.Context, code string, id string) (bool, error) {
	res, err := r.Lookup(ctx, discoverymodel.LookupRequest{
		Code:   code,
		Search: id,
		Limit:  100,
	})
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

func (r *Registry) ListResources() []ResourceDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ResourceDescriptor, 0, len(r.resources))
	for _, desc := range r.resources {
		items = append(items, desc)
	}
	sort.Slice(items, func(i, j int) bool {
		return resourceKey(items[i].Group, items[i].Resource) < resourceKey(items[j].Group, items[j].Resource)
	})
	return items
}

func (r *Registry) ListActions() []ActionDescriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := make([]ActionDescriptor, 0, len(r.actions))
	for _, desc := range r.actions {
		items = append(items, desc)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].ID < items[j].ID
	})
	return items
}

func (r *Registry) GetResource(group string, resource string) (ResourceDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	desc, ok := r.resources[resourceKey(group, resource)]
	return desc, ok
}

func (r *Registry) GetAction(id string) (ActionDescriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	desc, ok := r.actions[id]
	return desc, ok
}

func (r *Registry) SuggestResources(ctx context.Context, prefix string) ([]discoverymodel.DiscoverResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	suggestions := make([]discoverymodel.DiscoverResult, 0)
	seen := make(map[string]struct{})
	prefixLower := strings.ToLower(prefix)

	for key := range r.resources {
		if strings.HasPrefix(key, prefixLower) {
			if _, exists := seen[key]; !exists {
				suggestions = append(suggestions, discoverymodel.DiscoverResult{
					FullID: key,
					Name:   key,
					Final:  false,
				})
				seen[key] = struct{}{}
			}
		}
	}

	for key, desc := range r.resources {
		if strings.HasPrefix(prefixLower, key+"/") && desc.DiscoverFunc != nil {
			remaining := prefix[len(key)+1:]
			matches, err := desc.DiscoverFunc(ctx, remaining, "", 100)
			if err != nil {
				continue
			}
			for _, item := range matches.Items {
				fullID := key + "/" + item.ID
				if _, exists := seen[fullID]; exists {
					continue
				}
				suggestions = append(suggestions, discoverymodel.DiscoverResult{
					FullID: fullID,
					Name:   item.Name,
					Final:  true,
				})
				seen[fullID] = struct{}{}
			}
			for _, wildcard := range []string{"*", "**"} {
				if remaining == "" || strings.HasPrefix(wildcard, strings.ToLower(remaining)) {
					fullID := key + "/" + wildcard
					if _, exists := seen[fullID]; exists {
						continue
					}
					suggestions = append(suggestions, discoverymodel.DiscoverResult{
						FullID: fullID,
						Name:   wildcard,
						Final:  true,
					})
					seen[fullID] = struct{}{}
				}
			}
		}
	}

	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].FullID < suggestions[j].FullID
	})
	return suggestions, nil
}

func (r *Registry) SuggestVerbs(_ context.Context, resourcePrefix string) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if resourcePrefix == "" {
		return []string{"*"}, nil
	}
	resourcePrefix = strings.ToLower(resourcePrefix)
	var bestMatch *ResourceDescriptor
	longestKey := 0
	for key, desc := range r.resources {
		if resourcePrefix == key || strings.HasPrefix(resourcePrefix, key+"/") {
			if len(key) > longestKey {
				descCopy := desc
				bestMatch = &descCopy
				longestKey = len(key)
			}
		}
	}
	if bestMatch == nil || len(bestMatch.Verbs) == 0 {
		return []string{"*"}, nil
	}
	return bestMatch.Verbs, nil
}

func (r *Registry) RegisterSAUsageChecker(checker func(ctx context.Context, id string) error) {
	if checker == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.saUsageCheckers = append(r.saUsageCheckers, checker)
}

func (r *Registry) CheckSAUsage(ctx context.Context, id string) error {
	r.mu.RLock()
	checkers := append([]func(ctx context.Context, id string) error(nil), r.saUsageCheckers...)
	r.mu.RUnlock()
	for _, checker := range checkers {
		if err := checker(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

func Paginate(items []discoverymodel.LookupItem, cursor string, limit int) *shared.PaginationResponse[discoverymodel.LookupItem] {
	total := len(items)
	if limit <= 0 {
		limit = 20
	}
	offset := 0
	if cursor != "" {
		if parsed, err := strconv.Atoi(cursor); err == nil {
			offset = parsed
		}
	}
	if offset >= total {
		return &shared.PaginationResponse[discoverymodel.LookupItem]{
			Items:      []discoverymodel.LookupItem{},
			NextCursor: "",
			HasMore:    false,
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
	return &shared.PaginationResponse[discoverymodel.LookupItem]{
		Items:      items[offset:end],
		NextCursor: nextCursor,
		HasMore:    hasMore,
	}
}

func normalizeStrings(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || slices.Contains(normalized, value) {
			continue
		}
		normalized = append(normalized, value)
	}
	return normalized
}

func resourceKey(group string, resource string) string {
	group = strings.Trim(strings.ToLower(group), "/")
	resource = strings.Trim(strings.ToLower(resource), "/")
	switch {
	case group == "" && resource == "":
		return ""
	case group == "":
		return resource
	case resource == "":
		return group
	default:
		return group + "/" + resource
	}
}
