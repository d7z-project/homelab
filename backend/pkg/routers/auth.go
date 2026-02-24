package routers

import (
	"context"
	"errors"
	"homelab/pkg/auth"
	"homelab/pkg/common"
	"net/http"

	"github.com/go-chi/render"
)

// LoginHandler godoc
// @Summary Login to the server
// @Description verify root password and return session id
// @Tags auth
// @Accept  json
// @Produce  json
// @Param request body auth.LoginRequest true "Login Request"
// @Success 200 {object} auth.LoginResponse
// @Failure 401 {object} common.Response
// @Router /login [post]
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req auth.LoginRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	sessionID, err := auth.Login(r.Context(), req.Password, req.Totp)
	if err != nil {
		if errors.Is(err, auth.ErrTotpRequired) {
			common.UnauthorizedError(w, r, 10001, "TOTP Required")
			return
		}
		if errors.Is(err, auth.ErrUnauthorized) {
			common.UnauthorizedError(w, r, 10000, "Unauthorized")
			return
		}
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(w, r, auth.LoginResponse{
		SessionID: sessionID,
	})
}

// LogoutHandler godoc
// @Summary Logout from the server
// @Description invalidate session id
// @Tags auth
// @Accept  json
// @Produce  json
// @Security ApiKeyAuth
// @Success 200 {object} common.Response
// @Router /logout [post]
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("Authorization")
	if len(sessionID) > 7 && sessionID[:7] == "Bearer " {
		sessionID = sessionID[7:]
	}
	if sessionID == "" {
		common.Success(w, r, nil)
		return
	}

	err := auth.Logout(r.Context(), sessionID)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, nil)
}

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

		// 1. Check if it's a Root/Human session (always has full access)
		isRoot, err := auth.Verify(r.Context(), token)
		if err == nil && isRoot {
			ctx := context.WithValue(r.Context(), auth.AuthContextKey, &auth.AuthContext{
				Type: "root",
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		// 2. Check if it's a ServiceAccount token
		saName, err := auth.GetTokenSA(r.Context(), token)
		if err == nil && saName != "" {
			ctx := context.WithValue(r.Context(), auth.AuthContextKey, &auth.AuthContext{
				Type: "sa",
				Name: saName,
			})
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		common.UnauthorizedError(w, r, 10000, "Unauthorized")
	})
}

type AuthInfo struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// InfoHandler godoc
// @Summary Get current auth info
// @Tags auth
// @Produce json
// @Success 200 {object} AuthInfo
// @Router /info [get]
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	ac := auth.FromContext(r.Context())
	if ac == nil {
		common.UnauthorizedError(w, r, 10000, "Unauthorized")
		return
	}

	common.Success(w, r, AuthInfo{
		Type: ac.Type,
		Name: ac.Name,
	})
}

func RequirePermission(verb string, resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ac := auth.FromContext(r.Context())
			if ac == nil {
				common.UnauthorizedError(w, r, 10000, "Unauthorized")
				return
			}

			if ac.Type == "root" {
				next.ServeHTTP(w, r)
				return
			}

			if ac.Name != "" {
				allowed, err := auth.Authorize(r.Context(), ac.Name, verb, resource)
				if err == nil && allowed {
					next.ServeHTTP(w, r)
					return
				}
			}

			common.UnauthorizedError(w, r, 10002, "Permission Denied")
		})
	}
}
