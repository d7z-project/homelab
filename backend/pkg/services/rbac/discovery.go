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

	// Register rbac resources with specific verbs
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
	}, []string{"get", "list", "create", "update", "delete", "simulate", "*"})
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

	// 1. 匹配根资源 (Root Resources)
	// 即使 prefix 包含 "/"，它也可能是一个根资源的前缀 (例如 "net" 匹配 "network/ip")
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
	// 如果 prefix 已经进入了某个根资源的命名空间 (例如 "network/dns/")
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

	// 寻找匹配的根资源
	// 可能是完全匹配 (例如 "network/dns") 或者是该路径下的子资源 (例如 "network/dns/**")
	var bestMatch *resourceInfo
	longestKey := 0

	for name, info := range discoveredResources {
		// 完全匹配或前缀匹配且后续为斜杠 (资源实例路径)
		if resourcePrefixLower == name || strings.HasPrefix(resourcePrefixLower, name+"/") {
			if len(name) > longestKey {
				infoCopy := info // Avoid closure issues if any
				bestMatch = &infoCopy
				longestKey = len(name)
			}
		}
	}

	if bestMatch != nil {
		return bestMatch.verbs, nil
	}

	// 如果没有任何匹配且包含斜杠，默认返回通配符以防阻塞
	return []string{"*"}, nil
}
