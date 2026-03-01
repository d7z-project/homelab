package controllers

import (
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	authservice "homelab/pkg/services/auth"
	rbacservice "homelab/pkg/services/rbac"
	"net/http"

	"github.com/go-chi/render"
)

// LoginHandler godoc
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	sessionID, err := authservice.Login(r.Context(), req.Password, req.Totp)
	if err != nil {
		if err == authservice.ErrTotpRequired {
			common.UnauthorizedError(w, r, 10001, "TOTP Required")
			return
		}
		if err == authservice.ErrUnauthorized {
			common.UnauthorizedError(w, r, 10000, "Unauthorized")
			return
		}
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	common.Success(w, r, models.LoginResponse{
		SessionID: sessionID,
	})
}

// LogoutHandler godoc
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	sessionID := r.Header.Get("Authorization")
	if len(sessionID) > 7 && sessionID[:7] == "Bearer " {
		sessionID = sessionID[7:]
	}
	if sessionID == "" {
		common.Success(w, r, nil)
		return
	}

	err := authservice.Logout(r.Context(), sessionID)
	if err != nil {
		common.InternalServerError(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	common.Success(w, r, nil)
}

type AuthInfo struct {
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// InfoHandler godoc
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	ac := commonauth.FromContext(r.Context())
	if ac == nil {
		common.UnauthorizedError(w, r, 10000, "Unauthorized")
		return
	}

	common.Success(w, r, AuthInfo{
		Type: ac.Type,
		Name: ac.Name,
	})
}

// SuggestResourcesHandler godoc
func SuggestResourcesHandler(w http.ResponseWriter, r *http.Request) {
	prefix := r.URL.Query().Get("prefix")

	suggestions, err := rbacservice.SuggestResources(r.Context(), prefix)
	if err != nil {
		common.InternalServerError(w, r, 10001, err.Error())
		return
	}

	common.Success(w, r, suggestions)
}
