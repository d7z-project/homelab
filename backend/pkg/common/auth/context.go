package auth

import (
	"context"
	"errors"
	"homelab/pkg/models"
)

var (
	ErrUnauthorized = errors.New("unauthorized")
	ErrTotpRequired = errors.New("totp required")
)

type AuthContext struct {
	Type      string // "root" or "sa"
	ID        string // ServiceAccount ID if Type is "sa"
	SessionID string // UUID for revocation check
}

type contextKey string

const (
	AuthContextKey        contextKey = "auth_context"
	PermissionsContextKey contextKey = "permissions_context"
)

func FromContext(ctx context.Context) *AuthContext {
	val, _ := ctx.Value(AuthContextKey).(*AuthContext)
	return val
}

func WithAuth(ctx context.Context, auth *AuthContext) context.Context {
	return context.WithValue(ctx, AuthContextKey, auth)
}

func PermissionsFromContext(ctx context.Context) *models.ResourcePermissions {
	val, _ := ctx.Value(PermissionsContextKey).(*models.ResourcePermissions)
	return val
}

func WithPermissions(ctx context.Context, perms *models.ResourcePermissions) context.Context {
	return context.WithValue(ctx, PermissionsContextKey, perms)
}
