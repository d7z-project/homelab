package routers

import (
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
