package controllers

import (
	"context"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	authservice "homelab/pkg/services/auth"
	"net/http"

	"github.com/go-chi/chi/v5"
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

	ip := GetIP(r)
	ua := r.UserAgent()

	// Inject logger for login phase
	ctx := context.WithValue(r.Context(), commonaudit.LoggerContextKey, &commonaudit.AuditLogger{
		Subject:   "anonymous",
		Resource:  "auth",
		IPAddress: ip,
		UserAgent: ua,
	})

	sessionID, err := authservice.Login(ctx, req.Password, req.Totp, ip, ua)
	if err != nil {
		if err == authservice.ErrTotpRequired {
			common.UnauthorizedError(w, r, 10001, "totp required")
			return
		}
		common.UnauthorizedError(w, r, 10000, err.Error())
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
	Type      string `json:"type"`
	ID        string `json:"id,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
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
		Type:      ac.Type,
		ID:        ac.ID,
		SessionID: ac.SessionID,
	})
}

// ListSessionsHandler godoc
// @Summary List all active root sessions
// @Tags auth
// @Produce json
// @Success 200 {array} models.Session
// @Security ApiKeyAuth
// @Router /auth/sessions [get]
func ListSessionsHandler(w http.ResponseWriter, r *http.Request) {
	res, err := authservice.ListSessions(r.Context())
	if err != nil {
		common.UnauthorizedError(w, r, http.StatusUnauthorized, err.Error())
		return
	}
	common.Success(w, r, res)
}

// RevokeSessionHandler godoc
// @Summary Revoke a session
// @Tags auth
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {string} string "success"
// @Security ApiKeyAuth
// @Router /auth/sessions/{id} [delete]
func RevokeSessionHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := authservice.RevokeSession(r.Context(), id); err != nil {
		common.UnauthorizedError(w, r, http.StatusUnauthorized, err.Error())
		return
	}
	common.Success(w, r, "success")
}
