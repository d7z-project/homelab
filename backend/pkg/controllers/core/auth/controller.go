package auth

import (
	"context"
	apiv1 "homelab/pkg/apis/core/auth/v1"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	controllercommon "homelab/pkg/controllers"
	"net/http"

	authservice "homelab/pkg/services/core/auth"

	"github.com/go-chi/chi/v5"
)

// LoginHandler godoc
// @Summary Login to get session
// @Tags auth
// @Accept json
// @Produce json
// @Param request body apiv1.LoginRequest true "Login Request"
// @Success 200 {object} apiv1.LoginResponse
// @Failure 401 {object} common.Response "Unauthorized"
// @Router /auth/login [post]
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	req, ok := controllercommon.BindRequest[apiv1.LoginRequest](w, r)
	if !ok {
		return
	}

	ip := common.GetIP(r)
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
		if err == commonauth.ErrTotpRequired {
			common.UnauthorizedError(w, r, 10001, "totp required")
			return
		}
		common.UnauthorizedError(w, r, 10000, err.Error())
		return
	}

	common.Success(w, r, toAPILoginResponse(sessionID))
}

// LogoutHandler godoc
// @Summary Logout
// @Tags auth
// @Produce json
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /auth/logout [post]
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	authHeader := r.Header.Get("Authorization")
	token := authHeader
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		token = authHeader[7:]
	}

	_ = authservice.Logout(r.Context(), token)
	common.Success(w, r, "success")
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
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /auth/info [get]
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

// ScanSessionsHandler godoc
// @Summary Scan all active root sessions
// @Tags auth
// @Produce json
// @Success 200 {array} apiv1.Session
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /auth/sessions [get]
func ScanSessionsHandler(w http.ResponseWriter, r *http.Request) {
	res, err := authservice.ScanSessions(r.Context())
	if err != nil {
		common.UnauthorizedError(w, r, http.StatusUnauthorized, err.Error())
		return
	}
	common.Success(w, r, toAPISessions(res))
}

// RevokeSessionHandler godoc
// @Summary Revoke a session
// @Tags auth
// @Produce json
// @Param id path string true "Session ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Session Not Found"
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
