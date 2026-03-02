package controllers

import (
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	authservice "homelab/pkg/services/auth"
	"net/http"

	"github.com/go-chi/render"
)

// LoginHandler godoc
// @Summary Login to get session
// @Tags auth
// @Accept json
// @Produce json
// @Param request body models.LoginRequest true "Login Request"
// @Success 200 {object} models.LoginResponse
// @Router /login [post]
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	sessionID, err := authservice.Login(r.Context(), req.Password, req.Totp)
	if err != nil {
		common.UnauthorizedError(w, r, http.StatusUnauthorized, err.Error())
		return
	}

	common.Success(w, r, &models.LoginResponse{SessionID: sessionID})
}

// LogoutHandler godoc
// @Summary Logout
// @Tags auth
// @Produce json
// @Success 200 {string} string "success"
// @Security ApiKeyAuth
// @Router /logout [post]
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}

	_ = authservice.Logout(r.Context(), token)
	common.Success(w, r, nil)
}

type AuthInfo struct {
	Type string `json:"type"`
	ID   string `json:"id,omitempty"`
}

// InfoHandler godoc
// @Summary Get current user info
// @Tags auth
// @Produce json
// @Success 200 {object} AuthInfo
// @Security ApiKeyAuth
// @Router /info [get]
func InfoHandler(w http.ResponseWriter, r *http.Request) {
	ac := commonauth.FromContext(r.Context())
	if ac == nil {
		common.UnauthorizedError(w, r, 10000, "Unauthorized")
		return
	}

	common.Success(w, r, AuthInfo{
		Type: ac.Type,
		ID:   ac.ID,
	})
}
