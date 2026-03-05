package rbac

import (
	"context"
	"homelab/pkg/models"
	"sort"
	"strings"
	"sync"
)

var (
	// Resource Discovery
	discoveredResources map[string]resourceInfo
	discoveryMu         sync.RWMutex
)

type resourceInfo struct {
	discover DiscoverFunc
	verbs    []string
}

// DiscoverFunc returns a list of matching resource paths based on the remaining prefix
type DiscoverFunc func(ctx context.Context, prefix string) ([]models.DiscoverResult, error)

func init() {
	discoveredResources = make(map[string]resourceInfo)

	standardVerbs := []string{"get", "list", "create", "update", "delete", "*"}

	// Register default internal resources
	RegisterResourceWithVerbs("rbac", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		subs := []string{"serviceaccounts", "roles", "rolebindings", "simulate"}
		res := make([]models.DiscoverResult, 0)
		for _, s := range subs {
			if strings.HasPrefix(s, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: s,
					Name:   s,
					Final:  true,
				})
			}
		}
		return res, nil
	}, standardVerbs)
}

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

	if !strings.Contains(prefix, "/") {
		var rootNames []string
		for name := range discoveredResources {
			if strings.HasPrefix(name, prefixLower) {
				rootNames = append(rootNames, name)
			}
		}
		sort.Strings(rootNames)

		for _, name := range rootNames {
			suggestions = append(suggestions, models.DiscoverResult{
				FullID: name,
				Name:   name,
				Final:  false,
			})
		}
		return suggestions, nil
	}

	parts := strings.SplitN(prefixLower, "/", 2)
	baseRes := parts[0]
	remaining := parts[1]

	if info, ok := discoveredResources[baseRes]; ok {
		matches, err := info.discover(ctx, remaining)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			m.FullID = baseRes + "/" + m.FullID
			if _, exists := seen[m.FullID]; !exists {
				suggestions = append(suggestions, m)
				seen[m.FullID] = struct{}{}
			}
		}

		// Ensure standard wildcards are present
		wildcards := []string{"*", "**"}
		for _, w := range wildcards {
			if remaining == "" || strings.HasPrefix(w, remaining) {
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

	// Split to get the root resource name
	baseRes := strings.ToLower(strings.Split(resourcePrefix, "/")[0])
	if info, ok := discoveredResources[baseRes]; ok {
		// Only if we recognize the root resource do we suggest its specific verbs
		return info.verbs, nil
	}

	// If the resource is unknown, we don't know the verbs
	return []string{}, nil
}
