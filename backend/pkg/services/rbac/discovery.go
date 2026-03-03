package rbac

import (
	"context"
	dnsrepo "homelab/pkg/repositories/dns"
	orchrepo "homelab/pkg/repositories/orchestration"
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
		// prefix is everything after "dns/"
		parts := strings.Split(prefix, "/")

		// Get all domains to match against the first part
		domains, _, err := dnsrepo.ListDomains(ctx, 0, 1000, "")
		if err != nil {
			return nil, err
		}

		var res []string
		for _, d := range domains {
			// Check if domain matches parts[0]
			if !strings.HasPrefix(d.Name, parts[0]) {
				continue
			}

			if len(parts) <= 1 {
				// Level 1: Suggest domains
				res = append(res, d.Name)
				res = append(res, d.Name+"/*")
				res = append(res, d.Name+"/**")
			} else {
				// Level 2 & 3: We have a full domain match, suggest records
				if d.Name != parts[0] {
					continue
				}

				records, _, err := dnsrepo.ListRecords(ctx, d.ID, 0, 1000, "")
				if err != nil {
					continue
				}

				for _, r := range records {
					// Check if record host matches parts[1]
					if !strings.HasPrefix(r.Name, parts[1]) {
						continue
					}

					if len(parts) <= 2 {
						// Level 2: Suggest hostnames
						res = append(res, d.Name+"/"+r.Name)
						res = append(res, d.Name+"/"+r.Name+"/*")
					} else {
						// Level 3: Suggest types
						if r.Name == parts[1] && strings.HasPrefix(r.Type, parts[2]) {
							res = append(res, d.Name+"/"+r.Name+"/"+r.Type)
						}
					}
				}
			}
		}

		return res, nil
	}, standardVerbs)

	RegisterResourceWithVerbs("orchestration", func(ctx context.Context, prefix string) ([]string, error) {
		// Suggest static sub-resources
		subs := []string{"workflows", "instances", "manifests", "probe"}
		var res []string
		for _, s := range subs {
			if strings.HasPrefix(s, prefix) {
				res = append(res, s)
			}
		}

		// Suggest specific workflows
		workflows, err := orchrepo.ListWorkflows(ctx)
		if err == nil {
			for _, wf := range workflows {
				if strings.HasPrefix(wf.ID, prefix) {
					res = append(res, wf.ID)
				}
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
func SuggestResources(ctx context.Context, prefix string) ([]string, error) {
	discoveryMu.RLock()
	defer discoveryMu.RUnlock()

	var suggestions []string
	seen := make(map[string]struct{})

	if !strings.Contains(prefix, "/") {
		for name := range discoveredResources {
			if strings.HasPrefix(name, prefix) {
				suggestions = append(suggestions, name)
				seen[name] = struct{}{}
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
			fullPath := baseRes + "/" + m
			if _, exists := seen[fullPath]; !exists {
				suggestions = append(suggestions, fullPath)
				seen[fullPath] = struct{}{}
			}
		}

		// Ensure standard wildcards are present
		wildcards := []string{"*", "**"}
		for _, w := range wildcards {
			if remaining == "" || strings.HasPrefix(w, remaining) {
				fullPath := baseRes + "/" + w
				if _, exists := seen[fullPath]; !exists {
					suggestions = append(suggestions, fullPath)
					seen[fullPath] = struct{}{}
				}
			}
		}
	}

	sort.Strings(suggestions)
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
	baseRes := strings.Split(resourcePrefix, "/")[0]
	if info, ok := discoveredResources[baseRes]; ok {
		// Only if we recognize the root resource do we suggest its specific verbs
		return info.verbs, nil
	}

	// If the resource is unknown or partially typed, only suggest "*"
	return []string{"*"}, nil
}
