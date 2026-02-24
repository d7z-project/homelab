package auth

import (
	"context"
	"errors"
	"homelab/pkg/common"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrTotpRequired = errors.New("totp required")
)

type LoginRequest struct {
	Password string `json:"password"`
	Totp     string `json:"totp"`
}

type LoginResponse struct {
	SessionID string `json:"session_id"`
}

func Login(ctx context.Context, password string, totpCode string) (string, error) {
	if common.Opts.TotpAuth != "" {
		if totpCode == "" {
			return "", ErrTotpRequired
		}
		if !totp.Validate(totpCode, common.Opts.TotpAuth) {
			return "", ErrUnauthorized
		}
	}

	if password != common.Opts.RootPassword {
		return "", ErrUnauthorized
	}

	sessionID := uuid.New().String()
	sessionCtx := common.DB.Child("auth", "sessions")
	// Store session with 24 hours TTL, value as []byte
	err := sessionCtx.Put(ctx, sessionID, "root", 24*time.Hour)
	if err != nil {
		return "", err
	}
	return sessionID, nil
}

func Verify(ctx context.Context, sessionID string) (bool, error) {
	sessionCtx := common.DB.Child("auth", "sessions")
	val, err := sessionCtx.Get(ctx, sessionID)
	if err != nil {
		return false, nil
	}
	return string(val) == "root", nil
}

func Logout(ctx context.Context, sessionID string) error {
	sessionCtx := common.DB.Child("auth", "sessions")
	_, err := sessionCtx.Delete(ctx, sessionID)
	return err
}

type contextKey string

const AuthContextKey contextKey = "auth_context"

type AuthContext struct {
	Type string // "root" or "sa"
	Name string // ServiceAccount name if Type is "sa"
}

func FromContext(ctx context.Context) *AuthContext {
	val, _ := ctx.Value(AuthContextKey).(*AuthContext)
	return val
}
