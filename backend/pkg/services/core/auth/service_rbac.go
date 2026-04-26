package auth

import (
	"context"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	"strings"

	rbacmodel "homelab/pkg/models/core/rbac"
)

func GetPermissions(ctx context.Context, saID, verb, resource string) (*rbacmodel.ResourcePermissions, error) {
	if saID == "root" {
		return &rbacmodel.ResourcePermissions{
			AllowedAll:  true,
			MatchedRule: &rbacmodel.PolicyRule{Resource: "*", Verbs: []string{"*"}},
		}, nil
	}

	// Check if SA is enabled
	if !IsSAEnabled(ctx, saID, "") {
		return &rbacmodel.ResourcePermissions{}, nil
	}

	rbs, err := rbacrepo.ScanAllRoleBindings(ctx)
	if err != nil {
		return nil, err
	}

	perms := &rbacmodel.ResourcePermissions{}

	for _, rb := range rbs {
		if rb.Meta.Enabled && rb.Meta.ServiceAccountID == saID {
			for _, roleID := range rb.Meta.RoleIDs {
				role, err := rbacrepo.GetCachedRole(ctx, roleID)
				if err != nil {
					continue
				}
				for _, rule := range role.Meta.Rules {
					// Check if any verb in this rule matches requested verb
					if matchVerb(rule.Verbs, verb) {
						res := rule.Resource

						// Case 1: Full Wildcard
						if res == "*" || res == "**" {
							perms.AllowedAll = true
							perms.MatchedRule = &rule
							return perms, nil
						}

						cleanedRes := res
						if strings.HasSuffix(cleanedRes, "/**") {
							cleanedRes = strings.TrimSuffix(cleanedRes, "/**")
						} else if strings.HasSuffix(cleanedRes, "/*") {
							cleanedRes = strings.TrimSuffix(cleanedRes, "/*")
						}

						// Case 2: Exact Match or Prefix Match (e.g., resource "network/dns/domain/example.com"
						// matches rule "network/dns/domain/*").
						if cleanedRes == resource || strings.HasPrefix(resource, cleanedRes+"/") {
							perms.AllowedAll = true
							perms.MatchedRule = &rule
							return perms, nil
						}

						// Case 3: Instance-scoped route access. Preserve the full resource path so
						// downstream service checks can still call IsAllowed("base/...") directly.
						if strings.HasPrefix(cleanedRes, resource+"/") {
							inst := cleanedRes
							if inst != "" && inst != "*" && inst != "**" {
								perms.AllowedInstances = append(perms.AllowedInstances, inst)
								// Note: We don't return early here as multiple rules might contribute to the instances list
								if perms.MatchedRule == nil {
									perms.MatchedRule = &rule
								}
							}
						}
					}
				}
			}
		}
	}

	return perms, nil
}

func matchVerb(list []string, item string) bool {
	for _, v := range list {
		if v == "*" || v == item {
			return true
		}
	}
	return false
}
