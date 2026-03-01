package audit

import (
	"context"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
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

// Helper for other services to get logger from context
func FromContext(ctx context.Context) *commonaudit.AuditLogger {
	return commonaudit.FromContext(ctx)
}
