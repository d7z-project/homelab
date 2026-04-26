package auth

import (
	"context"
	"errors"

	rbacmodel "homelab/pkg/models/core/rbac"
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

func PermissionsFromContext(ctx context.Context) *rbacmodel.ResourcePermissions {
	val, ok := ctx.Value(PermissionsContextKey).(*rbacmodel.ResourcePermissions)
	if !ok || val == nil {
		return &rbacmodel.ResourcePermissions{}
	}
	return val
}

func WithPermissions(ctx context.Context, perms *rbacmodel.ResourcePermissions) context.Context {
	return context.WithValue(ctx, PermissionsContextKey, perms)
}

func WithIdentity(ctx context.Context, auth *AuthContext, perms *rbacmodel.ResourcePermissions) context.Context {
	ctx = WithAuth(ctx, auth)
	if perms != nil {
		ctx = WithPermissions(ctx, perms)
	}
	return ctx
}

func WithRoot(ctx context.Context) context.Context {
	return WithIdentity(ctx, &AuthContext{Type: "root"}, &rbacmodel.ResourcePermissions{AllowedAll: true})
}

func WithSystemSA(ctx context.Context) context.Context {
	return WithIdentity(ctx, &AuthContext{Type: "sa", ID: "system"}, &rbacmodel.ResourcePermissions{AllowedAll: true})
}

func RootContext() context.Context {
	ctx := context.Background()
	ctx = WithRoot(ctx)
	return ctx
}
