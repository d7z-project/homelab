package auth

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	authrepo "homelab/pkg/repositories/auth"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
)

func getSessionTTL() time.Duration {
	ttl, err := time.ParseDuration(common.Opts.SessionTTL)
	if err != nil {
		return 30 * time.Minute
	}
	return ttl
}

func Login(ctx context.Context, password, totpCode string, ip string, ua string) (string, error) {
	audit := commonaudit.FromContext(ctx)
	if common.Opts.TotpAuth != "" {
		if totpCode == "" {
			return "", ErrTotpRequired
		}
		if !totp.Validate(totpCode, common.Opts.TotpAuth) {
			audit.Log("Login", "root", "Login failed: invalid TOTP", false)
			return "", ErrUnauthorized
		}
	}

	if password != common.Opts.RootPassword {
		audit.Log("Login", "root", "Login failed: invalid password", false)
		return "", ErrUnauthorized
	}

	sessionID := uuid.New().String()
	err := authrepo.SaveSession(ctx, sessionID, "root", ip, ua, getSessionTTL())
	if err != nil {
		audit.Log("Login", "root", "Login failed: "+err.Error(), false)
		return "", err
	}

	// Generate JWT
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "root",
		"jti": sessionID,
		"iat": time.Now().Unix(),
	})

	tokenString, err := token.SignedString([]byte(common.Opts.JWTSecret))
	if err != nil {
		authrepo.DeleteSession(ctx, sessionID)
		audit.Log("Login", "root", "Login failed: jwt error", false)
		return "", err
	}

	audit.Log("Login", "root", fmt.Sprintf("Login successful from %s", ip), true)
	return tokenString, nil
}

func Verify(ctx context.Context, tokenString string, currentIP string, currentUA string) (bool, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(common.Opts.JWTSecret), nil
	})

	if err != nil || !token.Valid {
		return false, nil
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return false, nil
	}

	sub, _ := claims["sub"].(string)
	if sub != "root" {
		return false, nil
	}

	jti, ok := claims["jti"].(string)
	if !ok {
		return false, nil
	}

	userType, storedIP, storedUA, err := authrepo.GetSession(ctx, jti)
	if err != nil {
		return false, nil
	}

	// IP/UA Validation
	if userType == "root" {
		if storedIP != "" && storedIP != currentIP {
			commonaudit.FromContext(ctx).Log("Security", jti, fmt.Sprintf("IP mismatch: %s -> %s", storedIP, currentIP), false)
			return false, nil
		}
		if storedUA != "" && storedUA != currentUA {
			commonaudit.FromContext(ctx).Log("Security", jti, "User-Agent mismatch", false)
			return false, nil
		}
		// Refresh session
		_ = authrepo.RefreshSession(ctx, jti, getSessionTTL())
	}

	return userType == "root", nil
}

func Logout(ctx context.Context, tokenString string) error {
	ac := commonauth.FromContext(ctx)
	if ac != nil {
		commonaudit.FromContext(ctx).Log("Logout", ac.ID, "Logout successful", true)
	}

	// Try to extract JTI from token to delete session
	token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(common.Opts.JWTSecret), nil
	})

	if token != nil {
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			if jti, ok := claims["jti"].(string); ok {
				return authrepo.DeleteSession(ctx, jti)
			}
		}
	}

	return nil
}

func ScanSessions(ctx context.Context) ([]models.Session, error) {
	ac := commonauth.FromContext(ctx)
	if ac == nil || ac.Type != "root" {
		return nil, fmt.Errorf("%w: session (root access required)", commonauth.ErrPermissionDenied)
	}
	return authrepo.ScanSessions(ctx)
}

func RevokeSession(ctx context.Context, sessionID string) error {
	ac := commonauth.FromContext(ctx)
	if ac == nil || ac.Type != "root" {
		return fmt.Errorf("%w: session (root access required)", commonauth.ErrPermissionDenied)
	}
	err := authrepo.RevokeSession(ctx, sessionID)
	message := "Revoked session " + sessionID
	commonaudit.FromContext(ctx).Log("RevokeSession", sessionID, message, err == nil)
	return err
}
