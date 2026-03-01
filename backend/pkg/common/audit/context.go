package audit

import (
	"context"
	"homelab/pkg/models"
	auditrepo "homelab/pkg/repositories/audit"
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
func (l *AuditLogger) Log(action string, targetID string, success bool) {
	if l == nil {
		return
	}
	status := "Success"
	if !success {
		status = "Failed"
	}
	logEntry := &models.AuditLog{
		Subject:   l.Subject,
		Action:    action,
		Resource:  l.Resource,
		TargetID:  targetID,
		Status:    status,
		IPAddress: l.IPAddress,
		UserAgent: l.UserAgent,
	}
	go func() {
		_ = auditrepo.SaveLog(context.Background(), logEntry)
	}()
}

// FromContext retrieves the AuditLogger from the context
func FromContext(ctx context.Context) *AuditLogger {
	val, _ := ctx.Value(LoggerContextKey).(*AuditLogger)
	return val
}
