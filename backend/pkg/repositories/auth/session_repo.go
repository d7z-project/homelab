package auth

import (
	"context"
	"homelab/pkg/common"
	"time"
)

func SaveSession(ctx context.Context, sessionID string, userType string, ttl time.Duration) error {
	db := common.DB.Child("auth", "sessions")
	return db.Put(ctx, sessionID, userType, ttl)
}

func GetSession(ctx context.Context, sessionID string) (string, error) {
	db := common.DB.Child("auth", "sessions")
	val, err := db.Get(ctx, sessionID)
	if err != nil {
		return "", err
	}
	return string(val), nil
}

func DeleteSession(ctx context.Context, sessionID string) error {
	db := common.DB.Child("auth", "sessions")
	_, err := db.Delete(ctx, sessionID)
	return err
}
