package controllers

import (
	"homelab/pkg/common"
	"homelab/pkg/controllers/middlewares"
	"homelab/pkg/models"
	auditservice "homelab/pkg/services/audit"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// ListAuditLogsHandler godoc
// @Summary List audit logs
// @Tags audit
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search query"
// @Success 200 {object} common.CursorResponse{items=[]models.AuditLog}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /audit/logs [get]
func ListAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")

	res, err := auditservice.ScanLogs(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}

	common.CursorSuccess(w, r, res)
}

// CleanupAuditLogsHandler godoc
// @Summary Cleanup old audit logs
// @Tags audit
// @Produce json
// @Param days query int true "Logs older than these days will be deleted"
// @Success 200 {object} models.AuditCleanupResponse
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
		HandleError(w, r, err)
		return
	}

	common.Success(w, r, &models.AuditCleanupResponse{Deleted: count})
}

// AuditRouter registers the audit routes
func AuditRouter(r chi.Router) {
	r.Route("/audit", func(r chi.Router) {
		r.With(middlewares.RequirePermission("list", "audit")).Get("/logs", ListAuditLogsHandler)
		r.With(middlewares.RequirePermission("delete", "audit")).Post("/logs/cleanup", CleanupAuditLogsHandler)
	})
}
