package audit

import (
	apiv1 "homelab/pkg/apis/core/audit/v1"
	"homelab/pkg/common"
	controllercommon "homelab/pkg/controllers"
	auditservice "homelab/pkg/services/core/audit"
	"net/http"
	"strconv"
)

// ScanAuditLogsHandler godoc
// @Summary Scan audit logs
// @Tags audit
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search query"
// @Success 200 {object} common.CursorResponse{items=[]apiv1.AuditLog}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /audit/logs [get]
func ScanAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit, search := controllercommon.GetSearchCursorParams(r)

	res, err := auditservice.ScanLogs(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}

	common.CursorSuccess(w, r, mapAuditLogs(res))
}

// CleanupAuditLogsHandler godoc
// @Summary Cleanup old audit logs
// @Tags audit
// @Produce json
// @Param days query int true "Logs older than these days will be deleted"
// @Success 200 {object} apiv1.AuditCleanupResponse
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /audit/logs/cleanup [post]
func CleanupAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	daysStr := r.URL.Query().Get("days")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 0 {
		common.BadRequestError(w, r, http.StatusBadRequest, "Invalid days parameter")
		return
	}

	count, err := auditservice.CleanupLogs(r.Context(), days)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}

	common.Success(w, r, &apiv1.AuditCleanupResponse{Deleted: count})
}
