package audit

import (
	"context"
	"errors"
	"fmt"

	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	auditrepo "homelab/pkg/repositories/core/audit"

	auditmodel "homelab/pkg/models/core/audit"
	"homelab/pkg/models/shared"
)

// ScanLogs retrieves audit logs with optional pagination and filtering.
func ScanLogs(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[auditmodel.AuditLog], error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("audit") {
		return nil, fmt.Errorf("%w: audit", commonauth.ErrPermissionDenied)
	}
	return auditrepo.ScanLogs(ctx, cursor, limit, search)
}

// CleanupLogs deletes logs older than specified days. Only root can call this.
func CleanupLogs(ctx context.Context, days int) (int, error) {
	ac := commonauth.FromContext(ctx)
	if ac == nil || ac.Type != "root" {
		return 0, fmt.Errorf("%w: audit (root access required)", commonauth.ErrPermissionDenied)
	}

	if days < 0 {
		return 0, errors.New("days must be non-negative")
	}

	count, err := auditrepo.CleanupLogs(ctx, days)

	message := fmt.Sprintf("Cleaned up %d audit logs older than %d days", count, days)
	if err != nil {
		commonaudit.FromContext(ctx).Log("CleanupLogs", "system/audit", message+" (Error: "+err.Error()+")", false)
		return count, err
	}

	commonaudit.FromContext(ctx).Log("CleanupLogs", "system/audit", message, true)
	return count, nil
}

// FromContext returns the audit logger from context for internal callers.
func FromContext(ctx context.Context) *commonaudit.AuditLogger {
	return commonaudit.FromContext(ctx)
}
