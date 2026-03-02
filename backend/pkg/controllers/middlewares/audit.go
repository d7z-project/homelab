package middlewares

import (
	"context"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"net/http"
)

// AuditMiddleware injects an AuditLogger into the request context.
func AuditMiddleware(resource string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Extract Subject (User)
			subject := "anonymous"
			if ac := commonauth.FromContext(r.Context()); ac != nil {
				subject = ac.Type
				if ac.ID != "" {
					subject = ac.ID
				}
			}

			// Get IP (Handle potential proxies)
			ip := r.Header.Get("X-Forwarded-For")
			if ip == "" {
				ip = r.Header.Get("X-Real-IP")
			}
			if ip == "" {
				ip = r.RemoteAddr
			}

			logger := &commonaudit.AuditLogger{
				Subject:   subject,
				Resource:  resource,
				IPAddress: ip,
				UserAgent: r.UserAgent(),
			}

			ctx := context.WithValue(r.Context(), commonaudit.LoggerContextKey, logger)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
