package auth

import (
	"context"
	"errors"
	"homelab/pkg/models"
)

var (
	ErrUnauthorized     = errors.New("unauthorized")
	ErrTotpRequired     = errors.New("totp required")
	ErrPermissionDenied = errors.New("permission denied")
)

type AuthContext struct {
	Type      string // "root" or "sa"
	ID        string // ServiceAccount ID if Type is "sa"
	SessionID string // UUID for root session
}

type contextKey string

const (
	AuthContextKey        contextKey = "auth"
	PermissionsContextKey contextKey = "permissions"
)

func FromContext(ctx context.Context) *AuthContext {
	val, ok := ctx.Value(AuthContextKey).(*AuthContext)
	if !ok {
		return nil
	}
	return val
}

func WithAuth(ctx context.Context, auth *AuthContext) context.Context {
	return context.WithValue(ctx, AuthContextKey, auth)
}

func PermissionsFromContext(ctx context.Context) *models.ResourcePermissions {
	val, ok := ctx.Value(PermissionsContextKey).(*models.ResourcePermissions)
	if !ok || val == nil {
		return &models.ResourcePermissions{}
	}
	return val
}

func WithPermissions(ctx context.Context, perms *models.ResourcePermissions) context.Context {
	return context.WithValue(ctx, PermissionsContextKey, perms)
}

// SystemContext returns a background context with root permissions.
func SystemContext() context.Context {
	ctx := context.Background()
	ctx = WithAuth(ctx, &AuthContext{Type: "root"})
	ctx = WithPermissions(ctx, &models.ResourcePermissions{AllowedAll: true})
	return ctx
}
