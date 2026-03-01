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
	Type string // "root" or "sa"
	Name string // ServiceAccount name if Type is "sa"
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

func PermissionsFromContext(ctx context.Context) *models.ResourcePermissions {
	val, _ := ctx.Value(PermissionsContextKey).(*models.ResourcePermissions)
	return val
}
