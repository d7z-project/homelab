package v1

import (
	"errors"
	"net/http"
)

type LoginRequest struct {
	Password string `json:"password"`
	Totp     string `json:"totp"`
}

func (r *LoginRequest) Bind(_ *http.Request) error {
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

type LoginResponse struct {
	SessionID string `json:"session_id"`
}

type Session struct {
	ID        string `json:"id"`
	UserType  string `json:"userType"`
	CreatedAt string `json:"createdAt"`
	IP        string `json:"ip"`
	UserAgent string `json:"userAgent"`
}
