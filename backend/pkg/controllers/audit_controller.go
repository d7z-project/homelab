package controllers

import (
	"homelab/pkg/common"
	auditservice "homelab/pkg/services/audit"
	"net/http"
	"strconv"
)

// ListAuditLogsHandler godoc
// @Summary List audit logs
// @Description Retrieve a paginated list of audit logs
// @Tags audit
// @Accept json
// @Produce json
// @Param page query int false "Page number (0-based)"
// @Param pageSize query int false "Number of items per page"
// @Param search query string false "Search query by subject, action or resource"
// @Success 200 {object} common.PaginatedResponse{items=[]models.AuditLog}
// @Router /audit/logs [get]
// @Security BearerAuth
func ListAuditLogsHandler(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 0 {
		page = 0
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if pageSize <= 0 {
		pageSize = 20
	}
	search := r.URL.Query().Get("search")

	res, err := auditservice.ListLogs(r.Context(), page, pageSize, search)
	if err != nil {
		common.InternalServerError(w, r, 10001, err.Error())
		return
	}

	common.Success(w, r, res)
}
