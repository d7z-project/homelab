package middlewares

import (
	"context"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	authservice "homelab/pkg/services/auth"
	"net/http"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		token := authHeader
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			token = authHeader[7:]
		}

		if token == "" {
			common.UnauthorizedError(w, r, 10000, "Unauthorized")
			return
		}

		// 1. Check if it's a Root/Human session
		isRoot, err := authservice.Verify(r.Context(), token)
		if err == nil && isRoot {
			ctx := context.WithValue(r.Context(), commonauth.AuthContextKey, &commonauth.AuthContext{
				Type: "root",
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// 2. Check if it's a ServiceAccount token
		saName, err := authservice.GetTokenSA(r.Context(), token)
		if err == nil && saName != "" {
			ctx := context.WithValue(r.Context(), commonauth.AuthContextKey, &commonauth.AuthContext{
				Type: "sa",
				Name: saName,
			})
			authservice.UpdateSALastUsed(saName)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
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

			if ac.Name != "" {
				perms, err := authservice.GetPermissions(r.Context(), ac.Name, verb, resource)
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
