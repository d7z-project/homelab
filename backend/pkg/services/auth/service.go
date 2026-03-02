package auth

import (
	"context"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	authrepo "homelab/pkg/repositories/auth"
	rbacrepo "homelab/pkg/repositories/rbac"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

var (
	ErrUnauthorized = commonauth.ErrUnauthorized
	ErrTotpRequired = commonauth.ErrTotpRequired
)

var saLastUsed common.SyncMap[string, time.Time]

func UpdateSALastUsed(saName string) {
	now := time.Now()
	if lastUpdate, ok := saLastUsed.Load(saName); ok {
		if now.Sub(lastUpdate) < 5*time.Minute {
			return // Skip if updated recently
		}
	}
	saLastUsed.Store(saName, now)

	go func() {
		ctx := context.Background()
		sa, err := rbacrepo.GetServiceAccount(ctx, saName)
		if err == nil && sa != nil {
			sa.LastUsedAt = now.Format(time.RFC3339)
			_ = rbacrepo.SaveServiceAccount(ctx, sa)
		}
	}()
}

func Login(ctx context.Context, password, totpCode string) (string, error) {
	if common.Opts.TotpAuth != "" {
		if totpCode == "" {
			return "", ErrTotpRequired
		}
		if !totp.Validate(totpCode, common.Opts.TotpAuth) {
			return "", ErrUnauthorized
		}
	}

	if password != common.Opts.RootPassword {
		return "", ErrUnauthorized
	}

	sessionID := uuid.New().String()
	err := authrepo.SaveSession(ctx, sessionID, "root", 24*time.Hour)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func Verify(ctx context.Context, sessionID string) (bool, error) {
	userType, err := authrepo.GetSession(ctx, sessionID)
	if err != nil {
		return false, nil
	}
	return userType == "root", nil
}

func Logout(ctx context.Context, sessionID string) error {
	return authrepo.DeleteSession(ctx, sessionID)
}

func GetTokenSA(ctx context.Context, token string) (string, error) {
	return rbacrepo.GetTokenSA(ctx, token)
}

func GetPermissions(ctx context.Context, saName, verb, resource string) (*models.ResourcePermissions, error) {
	if saName == "root" {
		return &models.ResourcePermissions{
			AllowedAll: true,
			MatchedRule: &models.PolicyRule{Resource: "*", Verbs: []string{"*"}},
		}, nil
	}

	rbs, err := rbacrepo.ListRoleBindingsAll(ctx)
	if err != nil {
		return nil, err
	}

	perms := &models.ResourcePermissions{}

	for _, rb := range rbs {
		if rb.Enabled && rb.ServiceAccountName == saName {
			for _, roleName := range rb.RoleNames {
				role, err := rbacrepo.GetRole(ctx, roleName)
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

						// Case 2: Exact Match or Prefix Match (e.g., resource "dns/a" matches rule "dns/*")
						if cleanedRes == resource || strings.HasPrefix(resource, cleanedRes+"/") {
							perms.AllowedAll = true
							perms.MatchedRule = &rule
							return perms, nil
						} 
						
						// Case 3: Instance Suggestion (e.g., resource "dns" matches rule "dns/a")
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
