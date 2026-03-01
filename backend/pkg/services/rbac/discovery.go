package rbac

import (
	"context"
	"sort"
	"strings"
	"sync"
)

var (
	// Resource Discovery
	discoveredResources map[string]DiscoverFunc
	discoveryMu         sync.RWMutex
)

// DiscoverFunc returns a list of matching resource paths based on the remaining prefix
type DiscoverFunc func(ctx context.Context, prefix string) ([]string, error)

func init() {
	discoveredResources = make(map[string]DiscoverFunc)

	// Register default internal resources
	RegisterResource("rbac", func(ctx context.Context, prefix string) ([]string, error) {
		subs := []string{"serviceaccounts", "roles", "rolebindings", "simulate"}
		var res []string
		for _, s := range subs {
			if strings.HasPrefix(s, prefix) {
				res = append(res, s)
			}
		}
		return res, nil
	})
	RegisterResource("audit", func(ctx context.Context, prefix string) ([]string, error) {
		subs := []string{"logs"}
		var res []string
		for _, s := range subs {
			if strings.HasPrefix(s, prefix) {
				res = append(res, s)
			}
		}
		return res, nil
	})
}

// RegisterResource allows modules to register their resource types and instance providers
func RegisterResource(name string, f DiscoverFunc) {
	discoveryMu.Lock()
	defer discoveryMu.Unlock()
	discoveredResources[name] = f
}

// SuggestResources returns a list of resource paths matching the prefix
func SuggestResources(ctx context.Context, prefix string) ([]string, error) {
	discoveryMu.RLock()
	defer discoveryMu.RUnlock()

	var suggestions []string

	if !strings.Contains(prefix, "/") {
		for name := range discoveredResources {
			if strings.HasPrefix(name, prefix) {
				suggestions = append(suggestions, name)
			}
		}
		sort.Strings(suggestions)
		return suggestions, nil
	}

	parts := strings.SplitN(prefix, "/", 2)
	baseRes := parts[0]
	remaining := parts[1]

	if f, ok := discoveredResources[baseRes]; ok {
		matches, err := f(ctx, remaining)
		if err != nil {
			return nil, err
		}
		for _, m := range matches {
			suggestions = append(suggestions, baseRes+"/"+m)
		}

		if remaining == "" || strings.HasPrefix("*", remaining) {
			suggestions = append(suggestions, baseRes+"/*")
		}
		if remaining == "" || strings.HasPrefix("**", remaining) {
			suggestions = append(suggestions, baseRes+"/**")
		}
	}

	sort.Strings(suggestions)
	return suggestions, nil
}
