package audit

import (
	"context"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	auditrepo "homelab/pkg/repositories/audit"
	"homelab/pkg/services/rbac"
	"strings"
)

func init() {
	rbac.RegisterResourceWithVerbs("audit", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {
		subs := []string{"logs"}
		res := make([]models.DiscoverResult, 0)
		for _, s := range subs {
			if strings.HasPrefix(s, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: s,
					Name:   s,
					Final:  true,
				})
			}
		}
		return res, nil
	}, []string{"get", "list", "*"})
}

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

// Helper for other services to get logger from context
func FromContext(ctx context.Context) *commonaudit.AuditLogger {
	return commonaudit.FromContext(ctx)
}
