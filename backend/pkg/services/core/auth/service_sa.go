package auth

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	rbacmodel "homelab/pkg/models/core/rbac"
	secretmodel "homelab/pkg/models/core/secret"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	secretservice "homelab/pkg/services/core/secret"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var saLastUsed common.SyncMap[string, time.Time]

func UpdateSALastUsed(saID string) {
	now := time.Now()
	if lastUpdate, ok := saLastUsed.Load(saID); ok {
		if now.Sub(lastUpdate) < 5*time.Minute {
			return // Skip if updated recently
		}
	}
	saLastUsed.Store(saID, now)

	go func() {
		ctx := context.Background()
		_ = rbacrepo.UpdateServiceAccountStatus(ctx, saID, func(status *rbacmodel.ServiceAccountV1Status) {
			status.LastUsedAt = now.Format(time.RFC3339)
		})
		_ = secretservice.Touch(ctx, secretmodel.OwnerKindServiceAccount, saID, secretmodel.PurposeAuthToken)
	}()
}

func CreateSAToken(saID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":   "sa",
		"sa_id": saID,
		"iat":   time.Now().Unix(),
		"jti":   uuid.New().String(),
	})
	return token.SignedString([]byte(common.Opts.JWTSecret))
}

func VerifySAToken(ctx context.Context, tokenString string) (string, error) {
	_ = ctx
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(common.Opts.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return "", nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", nil
	}

	sub, _ := claims["sub"].(string)
	if sub != "sa" {
		return "", nil
	}

	saID, _ := claims["sa_id"].(string)
	return saID, nil
}

func IsSAEnabled(ctx context.Context, saID string, currentToken string) bool {
	sa, err := rbacrepo.GetServiceAccount(ctx, saID)
	if err != nil || sa == nil {
		return false
	}
	if !sa.Meta.Enabled || !sa.Status.HasAuthSecret {
		return false
	}
	if currentToken != "" {
		if !secretservice.Matches(ctx, secretmodel.OwnerKindServiceAccount, saID, secretmodel.PurposeAuthToken, currentToken) {
			return false
		}
	}
	return true
}
