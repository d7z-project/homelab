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
		return &models.ResourcePermissions{AllowedAll: true}, nil
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
					if match(rule.Verbs, verb) {
						for _, res := range rule.Resources {
							if res == "*" {
								perms.AllowedAll = true
								continue
							}

							cleanedRes := res
							if strings.HasSuffix(cleanedRes, "/**") {
								cleanedRes = strings.TrimSuffix(cleanedRes, "/**")
							} else if strings.HasSuffix(cleanedRes, "/*") {
								cleanedRes = strings.TrimSuffix(cleanedRes, "/*")
							}

							if cleanedRes == resource || strings.HasPrefix(resource, cleanedRes+"/") {
								perms.AllowedAll = true
							} else if strings.HasPrefix(cleanedRes, resource+"/") {
								inst := strings.TrimPrefix(cleanedRes, resource+"/")
								if inst != "" && inst != "*" && inst != "**" {
									perms.AllowedInstances = append(perms.AllowedInstances, inst)
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

func match(list []string, item string) bool {
	for _, v := range list {
		if v == "*" || v == item {
			return true
		}
	}
	return false
}
