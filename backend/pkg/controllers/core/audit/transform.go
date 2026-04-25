package audit

import (
	apiv1 "homelab/pkg/apis/core/audit/v1"
	auditmodel "homelab/pkg/models/core/audit"
	"homelab/pkg/models/shared"
)

func toAPIAuditLog(model auditmodel.AuditLog) apiv1.AuditLog {
	return apiv1.AuditLog{
		ID:        model.ID,
		Timestamp: model.Timestamp,
		Subject:   model.Subject,
		Action:    model.Action,
		Resource:  model.Resource,
		TargetID:  model.TargetID,
		Message:   model.Message,
		Status:    model.Status,
		IPAddress: model.IPAddress,
		UserAgent: model.UserAgent,
	}
}

func mapAuditLogs(res *shared.PaginationResponse[auditmodel.AuditLog]) *shared.PaginationResponse[apiv1.AuditLog] {
	items := make([]apiv1.AuditLog, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, toAPIAuditLog(item))
	}
	return &shared.PaginationResponse[apiv1.AuditLog]{Items: items, NextCursor: res.NextCursor, HasMore: res.HasMore}
}
