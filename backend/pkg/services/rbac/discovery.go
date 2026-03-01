package rbac

import (
	"context"
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
type DiscoverFunc func(ctx context.Context, prefix string) ([]string, error)

func init() {
	discoveredResources = make(map[string]resourceInfo)

	standardVerbs := []string{"get", "list", "create", "update", "delete", "*"}

	// Register default internal resources
	RegisterResourceWithVerbs("rbac", func(ctx context.Context, prefix string) ([]string, error) {
		subs := []string{"serviceaccounts", "roles", "rolebindings", "simulate"}
		var res []string
		for _, s := range subs {
			if strings.HasPrefix(s, prefix) {
				res = append(res, s)
			}
		}
		return res, nil
	}, standardVerbs)

	RegisterResourceWithVerbs("audit", func(ctx context.Context, prefix string) ([]string, error) {
		subs := []string{"logs"}
		var res []string
		for _, s := range subs {
			if strings.HasPrefix(s, prefix) {
				res = append(res, s)
			}
		}
		return res, nil
	}, []string{"get", "list", "*"})

	RegisterResourceWithVerbs("dns", func(ctx context.Context, prefix string) ([]string, error) {
		// This could be further expanded to list actual domains from repo
		// For now provide basic pattern suggestions
		return []string{}, nil
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

	if info, ok := discoveredResources[baseRes]; ok {
		matches, err := info.discover(ctx, remaining)
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

// SuggestVerbs returns supported verbs for a given resource prefix
func SuggestVerbs(ctx context.Context, resourcePrefix string) ([]string, error) {
	discoveryMu.RLock()
	defer discoveryMu.RUnlock()

	baseRes := strings.Split(resourcePrefix, "/")[0]
	if info, ok := discoveredResources[baseRes]; ok {
		return info.verbs, nil
	}

	return []string{"get", "list", "create", "update", "delete", "*"}, nil
}
