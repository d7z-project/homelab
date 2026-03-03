package auth

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"
	"time"
)

func SaveSession(ctx context.Context, sessionID string, userType string, ip string, ua string, ttl time.Duration) error {
	db := common.DB.Child("auth", "sessions")
	// Store as userType|createdAt|ip|ua
	val := fmt.Sprintf("%s|%s|%s|%s", userType, time.Now().Format(time.RFC3339), ip, ua)
	return db.Put(ctx, sessionID, val, ttl)
}

func GetSession(ctx context.Context, sessionID string) (userType string, ip string, ua string, err error) {
	db := common.DB.Child("auth", "sessions")
	val, err := db.Get(ctx, sessionID)
	if err != nil {
		return "", "", "", err
	}
	parts := strings.Split(string(val), "|")
	userType = parts[0]
	if len(parts) > 2 {
		ip = parts[2]
	}
	if len(parts) > 3 {
		ua = parts[3]
	}
	return userType, ip, ua, nil
}

func ListSessions(ctx context.Context) ([]models.Session, error) {
	db := common.DB.Child("auth", "sessions")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	var res []models.Session
	for _, item := range items {
		parts := strings.Split(item.Value, "|")
		userType := parts[0]
		createdAt := ""
		ip := ""
		ua := ""
		if len(parts) > 1 {
			createdAt = parts[1]
		}
		if len(parts) > 2 {
			ip = parts[2]
		}
		if len(parts) > 3 {
			ua = parts[3]
		}
		res = append(res, models.Session{
			ID:        item.Key,
			UserType:  userType,
			CreatedAt: createdAt,
			IP:        ip,
			UserAgent: ua,
		})
	}
	return res, nil
}

func RevokeSession(ctx context.Context, sessionID string) error {
	db := common.DB.Child("auth", "sessions")
	_, err := db.Delete(ctx, sessionID)
	return err
}

func RefreshSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	db := common.DB.Child("auth", "sessions")
	val, err := db.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	return db.Put(ctx, sessionID, string(val), ttl)
}

func DeleteSession(ctx context.Context, sessionID string) error {
	return RevokeSession(ctx, sessionID)
}
