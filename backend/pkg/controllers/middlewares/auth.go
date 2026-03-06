package middlewares

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	authservice "homelab/pkg/services/auth"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := common.GetIP(r)
		ua := r.UserAgent()

		// Inject a temporary logger for auth phase
		authLogger := &commonaudit.AuditLogger{
			Subject:   "anonymous",
			Resource:  "auth",
			IPAddress: ip,
			UserAgent: ua,
		}
		r = r.WithContext(context.WithValue(r.Context(), commonaudit.LoggerContextKey, authLogger))

		authHeader := r.Header.Get("Authorization")
		token := authHeader
		if len(authHeader) > 7 && strings.HasPrefix(authHeader, "Bearer ") {
			token = authHeader[7:]
		}

		if token == "" {
			common.UnauthorizedError(w, r, 10000, "Unauthorized")
			return
		}

		// Parse JWT
		jwtToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return []byte(common.Opts.JWTSecret), nil
		})

		if err == nil && jwtToken.Valid {
			if claims, ok := jwtToken.Claims.(jwt.MapClaims); ok {
				sub, _ := claims["sub"].(string)

				if sub == "root" {
					// 1. Root/Human session: Needs IP/UA and Revocation check
					jti, _ := claims["jti"].(string)
					isRoot, _ := authservice.Verify(r.Context(), token, ip, ua)
					if isRoot {
						// 1. Inject Identity
						ctx := context.WithValue(r.Context(), commonauth.AuthContextKey, &commonauth.AuthContext{
							Type:      "root",
							SessionID: jti,
						})
						// 2. Inject Global Permissions (important for manual service-layer checks)
						perms := &models.ResourcePermissions{
							AllowedAll: true,
						}
						ctx = context.WithValue(ctx, commonauth.PermissionsContextKey, perms)

						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				} else if sub == "sa" {
					// 2. ServiceAccount: Needs Enabled check and Token match, no IP/UA check
					saID, _ := claims["sa_id"].(string)
					if saID != "" && authservice.IsSAEnabled(r.Context(), saID, token) {
						// Load full permissions for this SA
						// Verb "*" and Resource "*" used here to load the generic permissions object
						perms, _ := authservice.GetPermissions(r.Context(), saID, "*", "*")

						ctx := context.WithValue(r.Context(), commonauth.AuthContextKey, &commonauth.AuthContext{
							Type: "sa",
							ID:   saID,
						})
						ctx = context.WithValue(ctx, commonauth.PermissionsContextKey, perms)

						authservice.UpdateSALastUsed(saID)
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					}
				}
			}
		}

		common.UnauthorizedError(w, r, 10000, "Unauthorized")
	})
}

func RequirePermission(verb string, resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac := commonauth.FromContext(r.Context())
			if ac == nil {
				common.UnauthorizedError(w, r, 10000, "Unauthorized")
				return
			}

			if ac.Type == "root" {
				perms := &models.ResourcePermissions{
					AllowedAll:  true,
					MatchedRule: &models.PolicyRule{Resource: "*", Verbs: []string{"*"}},
				}
				w.Header().Set("X-Matched-Policy", "*:*")
				ctx := context.WithValue(r.Context(), commonauth.PermissionsContextKey, perms)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if ac.ID != "" {
				perms, err := authservice.GetPermissions(r.Context(), ac.ID, verb, resource)
				if err == nil && perms != nil && (perms.AllowedAll || len(perms.AllowedInstances) > 0) {
					if perms.MatchedRule != nil {
						w.Header().Set("X-Matched-Policy", perms.MatchedRule.Resource)
					}
					ctx := context.WithValue(r.Context(), commonauth.PermissionsContextKey, perms)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}

			common.UnauthorizedError(w, r, 10002, "Permission Denied")
		})
	}
}
