package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	authrepo "homelab/pkg/repositories/auth"
	rbacrepo "homelab/pkg/repositories/rbac"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

var (
	ErrUnauthorized = commonauth.ErrUnauthorized
	ErrTotpRequired = commonauth.ErrTotpRequired
)

var saLastUsed common.SyncMap[string, time.Time]

func getSessionTTL() time.Duration {
	ttl, err := time.ParseDuration(common.Opts.SessionTTL)
	if err != nil {
		return 30 * time.Minute
	}
	return ttl
}

func UpdateSALastUsed(saID string) {
	now := time.Now()
	if lastUpdate, ok := saLastUsed.Load(saID); ok {
		if now.Sub(lastUpdate) < 5*time.Minute {
			return // Skip if updated recently
		}
	}
	saLastUsed.Store(saID, now)

	go func() {
		ctx := context.Background()
		sa, err := rbacrepo.GetServiceAccount(ctx, saID)
		if err == nil && sa != nil {
			sa.LastUsedAt = now.Format(time.RFC3339)
			_ = rbacrepo.SaveServiceAccount(ctx, sa)
		}
	}()
}

func Login(ctx context.Context, password, totpCode string, ip string, ua string) (string, error) {
	audit := commonaudit.FromContext(ctx)
	if common.Opts.TotpAuth != "" {
		if totpCode == "" {
			return "", ErrTotpRequired
		}
		if !totp.Validate(totpCode, common.Opts.TotpAuth) {
			audit.Log("Login", "root", "Login failed: invalid TOTP", false)
			return "", ErrUnauthorized
		}
	}

	if password != common.Opts.RootPassword {
		audit.Log("Login", "root", "Login failed: invalid password", false)
		return "", ErrUnauthorized
	}

	sessionID := uuid.New().String()
	err := authrepo.SaveSession(ctx, sessionID, "root", ip, ua, getSessionTTL())
	if err != nil {
		audit.Log("Login", "root", "Login failed: "+err.Error(), false)
		return "", err
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "root",
		"jti": sessionID,
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(common.Opts.JWTSecret))
	if err != nil {
		authrepo.DeleteSession(ctx, sessionID)
		audit.Log("Login", "root", "Login failed: jwt error", false)
		return "", err
	}

	audit.Log("Login", "root", fmt.Sprintf("Login successful from %s", ip), true)
	return tokenString, nil
}

func Verify(ctx context.Context, tokenString string, currentIP string, currentUA string) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(common.Opts.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return false, nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, nil
	}

	sub, _ := claims["sub"].(string)
	if sub != "root" {
		return false, nil
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		return false, nil
	}

	userType, storedIP, storedUA, err := authrepo.GetSession(ctx, jti)
	if err != nil {
		return false, nil
	}

	// IP/UA Validation
	if userType == "root" {
		if storedIP != "" && storedIP != currentIP {
			commonaudit.FromContext(ctx).Log("Security", jti, fmt.Sprintf("IP mismatch: %s -> %s", storedIP, currentIP), false)
			return false, nil
		}
		if storedUA != "" && storedUA != currentUA {
			commonaudit.FromContext(ctx).Log("Security", jti, "User-Agent mismatch", false)
			return false, nil
		}
		// Refresh session
		_ = authrepo.RefreshSession(ctx, jti, getSessionTTL())
	}

	return userType == "root", nil
}

func Logout(ctx context.Context, tokenString string) error {
	ac := commonauth.FromContext(ctx)
	if ac != nil {
		commonaudit.FromContext(ctx).Log("Logout", ac.ID, "Logout successful", true)
	}

	// Try to extract JTI from token to delete session
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(common.Opts.JWTSecret), nil
	})

	if token != nil {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if jti, ok := claims["jti"].(string); ok {
				return authrepo.DeleteSession(ctx, jti)
			}
		}
	}

	return nil
}

func ListSessions(ctx context.Context) ([]models.Session, error) {
	ac := commonauth.FromContext(ctx)
	if ac == nil || ac.Type != "root" {
		return nil, fmt.Errorf("%w: session (root access required)", commonauth.ErrPermissionDenied)
	}
	return authrepo.ListSessions(ctx)
}

func RevokeSession(ctx context.Context, sessionID string) error {
	ac := commonauth.FromContext(ctx)
	if ac == nil || ac.Type != "root" {
		return fmt.Errorf("%w: session (root access required)", commonauth.ErrPermissionDenied)
	}
	err := authrepo.RevokeSession(ctx, sessionID)
	message := "Revoked session " + sessionID
	commonaudit.FromContext(ctx).Log("RevokeSession", sessionID, message, err == nil)
	return err
}

func CreateSAToken(saID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "sa",
		"sa_id": saID,
		"iat":   time.Now().Unix(),
		"jti":   uuid.New().String(),
	})
	return token.SignedString([]byte(common.Opts.JWTSecret))
}

func VerifySAToken(ctx context.Context, tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(common.Opts.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return "", nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil
	}

	sub, _ := claims["sub"].(string)
	if sub != "sa" {
		return "", nil
	}

	saID, _ := claims["sa_id"].(string)
	return saID, nil
}

func HashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

func IsSAEnabled(ctx context.Context, saID string, currentToken string) bool {
	sa, err := rbacrepo.GetServiceAccount(ctx, saID)
	if err != nil || sa == nil {
		return false
	}
	// If currentToken is provided, it MUST match the hash stored in DB.
	if currentToken != "" {
		if sa.Token != HashToken(currentToken) {
			return false
		}
	}
	return sa.Enabled
}

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

	rbs, err := rbacrepo.ListRoleBindingsAll(ctx)
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
