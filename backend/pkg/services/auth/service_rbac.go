package auth

import (
	"context"
	"homelab/pkg/models"
	rbacrepo "homelab/pkg/repositories/rbac"
	"strings"
)

func GetPermissions(ctx context.Context, saID, verb, resource string) (*models.ResourcePermissions, error) {
	if saID == "root" {
		return &models.ResourcePermissions{
			AllowedAll:  true,
			MatchedRule: &models.PolicyRule{Resource: "*", Verbs: []string{"*"}},
		}, nil
	}

	// Check if SA is enabled
	if !IsSAEnabled(ctx, saID, "") {
		return &models.ResourcePermissions{}, nil
	}

	rbs, err := rbacrepo.ScanAllRoleBindings(ctx)
	if err != nil {
		return nil, err
	}

	perms := &models.ResourcePermissions{}

	for _, rb := range rbs {
		if rb.Enabled && rb.ServiceAccountID == saID {
			for _, roleID := range rb.RoleIDs {
				role, err := rbacrepo.GetRole(ctx, roleID)
				if err != nil {
					continue
				}
				for _, rule := range role.Rules {
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

						// Case 2: Exact Match or Prefix Match (e.g., resource "dns/a" matches rule "network/dns/*")
						if cleanedRes == resource || strings.HasPrefix(resource, cleanedRes+"/") {
							perms.AllowedAll = true
							perms.MatchedRule = &rule
							return perms, nil
						}

						// Case 3: Instance Suggestion (e.g., resource "network/dns" matches rule "dns/a")
						if strings.HasPrefix(cleanedRes, resource+"/") {
							inst := strings.TrimPrefix(cleanedRes, resource+"/")
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
