package audit

import (
	"context"

	"homelab/pkg/common/contextx"
	auditmodel "homelab/pkg/models/core/audit"
	auditrepo "homelab/pkg/repositories/core/audit"
)

type contextKey string

const LoggerContextKey contextKey = "audit_logger"

// AuditLogger is a tool class injected into the request context
type AuditLogger struct {
	Subject   string
	Resource  string
	IPAddress string
	UserAgent string
}

// Log an action manually (e.g. action="CreateServiceAccount", targetID="test-sa")
func (l *AuditLogger) Log(action string, targetID string, message string, success bool) {
	if l == nil {
		return
	}
	status := "Success"
	if !success {
		status = "Failed"
	}
	logEntry := &auditmodel.AuditLog{
		Subject:   l.Subject,
		Action:    action,
		Resource:  l.Resource,
		TargetID:  targetID,
		Message:   message,
		Status:    status,
		IPAddress: l.IPAddress,
		UserAgent: l.UserAgent,
	}
	go func() {
		_ = auditrepo.SaveLog(context.Background(), logEntry)
	}()
}

func WithLogger(ctx context.Context, logger *AuditLogger) context.Context {
	return contextx.WithValue(ctx, LoggerContextKey, logger)
}

// FromContext retrieves the AuditLogger from the context
func FromContext(ctx context.Context) *AuditLogger {
	val, _ := contextx.Value[*AuditLogger](ctx, LoggerContextKey)
	return val
}
