package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	authmodel "homelab/pkg/models/core/auth"

	"gopkg.d7z.net/middleware/kv"
)

var sessionDB kv.KV

func Configure(db kv.KV) {
	sessionDB = db
}

func SaveSession(ctx context.Context, sessionID string, userType string, ip string, ua string, ttl time.Duration) error {
	db := sessionDB.Child("auth", "sessions")
	// Store as userType|createdAt|ip|ua
	val := fmt.Sprintf("%s|%s|%s|%s", userType, time.Now().Format(time.RFC3339), ip, ua)
	return db.Put(ctx, sessionID, val, ttl)
}

func GetSession(ctx context.Context, sessionID string) (userType string, ip string, ua string, err error) {
	db := sessionDB.Child("auth", "sessions")
	val, err := db.Get(ctx, sessionID)
	if err != nil {
		return "", "", "", err
	}
	parts := strings.Split(val, "|")
	userType = parts[0]
	if len(parts) > 2 {
		ip = parts[2]
	}
	if len(parts) > 3 {
		ua = parts[3]
	}
	return userType, ip, ua, nil
}

func ScanSessions(ctx context.Context) ([]authmodel.Session, error) {
	db := sessionDB.Child("auth", "sessions")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	res := make([]authmodel.Session, 0)
	for _, v := range items {
		parts := strings.Split(v.Value, "|")
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
		res = append(res, authmodel.Session{
			ID:        v.Key,
			UserType:  userType,
			CreatedAt: createdAt,
			IP:        ip,
			UserAgent: ua,
		})
	}
	return res, nil
}

func RevokeSession(ctx context.Context, sessionID string) error {
	db := sessionDB.Child("auth", "sessions")
	_, err := db.Delete(ctx, sessionID)
	return err
}

func RefreshSession(ctx context.Context, sessionID string, ttl time.Duration) error {
	db := sessionDB.Child("auth", "sessions")
	val, err := db.Get(ctx, sessionID)
	if err != nil {
		return err
	}
	return db.Put(ctx, sessionID, val, ttl)
}

func DeleteSession(ctx context.Context, sessionID string) error {
	return RevokeSession(ctx, sessionID)
}
