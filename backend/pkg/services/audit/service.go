package audit

import (
	"context"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	auditrepo "homelab/pkg/repositories/audit"
)

// ListLogs retrieves audit logs with optional pagination and filtering.
func ListLogs(ctx context.Context, page, pageSize int, search string) (*common.PaginatedResponse, error) {
	logs, total, err := auditrepo.ListLogs(ctx, page, pageSize, search)
	if err != nil {
		return nil, err
	}

	var items []interface{}
	for _, l := range logs {
		items = append(items, l)
	}

	return &common.PaginatedResponse{
		Items: items,
		Total: total,
		Page:  page,
	}, nil
}

// CleanupLogs deletes logs older than specified days. Only root can call this.
func CleanupLogs(ctx context.Context, days int) (int, error) {
	ac := commonauth.FromContext(ctx)
	if ac == nil || ac.Type != "root" {
		return 0, errors.New("only root can cleanup audit logs")
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

// Helper for other services to get logger from context
func FromContext(ctx context.Context) *commonaudit.AuditLogger {
	return commonaudit.FromContext(ctx)
}
