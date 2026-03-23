package controllers

import (
	"fmt"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/controllers/middlewares"
	"homelab/pkg/models"
	siteservice "homelab/pkg/services/site"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

var sitePoolService *siteservice.SitePoolService
var siteAnalysisEngine *siteservice.AnalysisEngine
var siteExportManager *siteservice.ExportManager

func InitSiteControllers(service *siteservice.SitePoolService, engine *siteservice.AnalysisEngine, em *siteservice.ExportManager) {
	sitePoolService = service
	siteAnalysisEngine = engine
	siteExportManager = em
}

// Site Pools

// ListSiteGroupsHandler godoc
// @Summary Scan all site groups
// @Tags network/site
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name"
// @Success 200 {object} common.CursorResponse{items=[]models.SiteGroup}
// @Router /network/site/pools [get]
func ScanSiteGroupsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := sitePoolService.ScanGroups(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateSiteGroupHandler godoc
// @Summary Create a site group
// @Tags network/site
// @Accept json
// @Produce json
// @Param group body models.SiteGroup true "Site Group"
// @Success 200 {object} models.SiteGroup
// @Router /network/site/pools [post]
func CreateSiteGroupHandler(w http.ResponseWriter, r *http.Request) {
	var group models.SiteGroup
	if err := render.Bind(r, &group); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := sitePoolService.CreateGroup(r.Context(), &group); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, group)
}

// DeleteSiteGroupHandler godoc
// @Summary Delete a site group
// @Tags network/site
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {string} string "success"
// @Router /network/site/pools/{id} [delete]
func DeleteSiteGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/site/"+id) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/site") {
		HandleError(w, r, fmt.Errorf("%w: network/site/%s", commonauth.ErrPermissionDenied, id))
		return
	}
	if err := sitePoolService.DeleteGroup(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// PreviewSitePoolHandler godoc
// @Summary Preview site pool data (cursor-based)
// @Tags network/site
// @Produce json
// @Param id path string true "Group ID"
// @Param cursor query string false "Byte offset cursor"
// @Param limit query int false "Number of entries to return"
// @Param search query string false "Search prefix or tag"
// @Success 200 {object} models.SitePoolPreviewResponse
// @Router /network/site/pools/{id}/preview [get]
func PreviewSitePoolHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/site/"+id) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/site") {
		HandleError(w, r, fmt.Errorf("%w: network/site/%s", commonauth.ErrPermissionDenied, id))
		return
	}
	cursor := r.URL.Query().Get("cursor")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	search := r.URL.Query().Get("search")
	res, err := sitePoolService.PreviewPool(r.Context(), id, cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// ManageSitePoolEntryHandler godoc
// @Summary Add or update an entry in site pool
// @Tags network/site
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param entry body models.SitePoolEntryRequest true "Site Entry"
// @Success 200 {string} string "success"
// @Router /network/site/pools/{id}/entries [post]
func ManageSitePoolEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/site/"+id) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/site") {
		HandleError(w, r, fmt.Errorf("%w: network/site/%s", commonauth.ErrPermissionDenied, id))
		return
	}
	var req models.SitePoolEntryRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := sitePoolService.ManagePoolEntry(r.Context(), id, &req, "add"); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// DeleteSitePoolEntryHandler godoc
// @Summary Delete an entry from site pool
// @Tags network/site
// @Produce json
// @Param id path string true "Group ID"
// @Param type query int true "Rule type"
// @Param value query string true "Rule value"
// @Success 200 {string} string "success"
// @Router /network/site/pools/{id}/entries [delete]
func DeleteSitePoolEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/site/"+id) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/site") {
		HandleError(w, r, fmt.Errorf("%w: network/site/%s", commonauth.ErrPermissionDenied, id))
		return
	}
	value := r.URL.Query().Get("value")

	t, _ := strconv.Atoi(r.URL.Query().Get("type"))
	req := models.SitePoolEntryRequest{Type: uint8(t), Value: value}
	if err := sitePoolService.ManagePoolEntry(r.Context(), id, &req, "delete"); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// Site Analysis

// SiteHitTestHandler godoc
// @Summary Perform site hit test
// @Tags network/site
// @Accept json
// @Produce json
// @Param request body object true "Hit test request"
// @Success 200 {object} models.SiteAnalysisResult
// @Router /network/site/analysis/hit-test [post]
func SiteHitTestHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain   string   `json:"domain"`
		GroupIDs []string `json:"groupIds"`
	}
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	res, err := siteAnalysisEngine.HitTest(r.Context(), req.Domain, req.GroupIDs)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// Site Exports

// ListSiteExportsHandler godoc
// @Summary Scan all site exports
// @Tags network/site
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name"
// @Success 200 {object} common.CursorResponse{items=[]models.SiteExport}
// @Router /network/site/exports [get]
func ScanSiteExportsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := sitePoolService.ScanExports(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateSiteExportHandler godoc
// @Summary Create a site export
// @Tags network/site
// @Accept json
// @Produce json
// @Param export body models.SiteExport true "Site Export"
// @Success 200 {object} models.SiteExport
// @Router /network/site/exports [post]
func CreateSiteExportHandler(w http.ResponseWriter, r *http.Request) {
	var export models.SiteExport
	if err := render.Bind(r, &export); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := sitePoolService.CreateExport(r.Context(), &export); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, export)
}

// UpdateSiteExportHandler godoc
// @Summary Update a site export
// @Tags network/site
// @Accept json
// @Produce json
// @Param id path string true "Export ID"
// @Param export body models.SiteExport true "Site Export"
// @Success 200 {object} models.SiteExport
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Export Not Found"
// @Security ApiKeyAuth
// @Router /network/site/exports/{id} [put]
func UpdateSiteExportHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var export models.SiteExport
	if err := render.Bind(r, &export); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	export.ID = id
	if err := sitePoolService.UpdateExport(r.Context(), &export); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, export)
}

// DeleteSiteExportHandler godoc
// @Summary Delete a site export
// @Tags network/site
// @Produce json
// @Param id path string true "Export ID"
// @Success 200 {string} string "success"
// @Router /network/site/exports/{id} [delete]
func DeleteSiteExportHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := sitePoolService.DeleteExport(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ListSiteExportTasksHandler godoc
// @Summary Scan all site export tasks
// @Tags network/site
// @Produce json
// @Success 200 {array} siteservice.ExportTask
// @Security ApiKeyAuth
// @Router /network/site/exports/tasks [get]
func ScanSiteExportTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks := siteExportManager.ScanTasks()

	common.Success(w, r, tasks)
}

// TriggerSiteExportHandler godoc
// @Summary Trigger site export
// @Tags network/site
// @Produce json
// @Param id path string true "Export ID"
// @Param format query string false "Format: text, json, yaml"
// @Success 200 {object} map[string]string
// @Router /network/site/exports/{id}/trigger [post]
func TriggerSiteExportHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "text"
	}
	taskID, err := siteExportManager.TriggerExport(r.Context(), id, format)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, map[string]string{"taskId": taskID})
}

// SiteExportTaskStatusHandler godoc
// @Summary Get site export task status
// @Tags network/site
// @Produce json
// @Param taskId path string true "Task ID"
// @Success 200 {object} siteservice.ExportTask
// @Router /network/site/exports/task/{taskId} [get]
func SiteExportTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	task := siteExportManager.GetTask(id)
	if task == nil {
		common.Error(w, r, http.StatusNotFound, http.StatusNotFound, "task not found")
		return
	}
	common.Success(w, r, task)
}

// CancelSiteExportTaskHandler godoc
// @Summary Cancel a site export task
// @Tags network/site
// @Produce json
// @Param taskId path string true "Task ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Task Not Found"
// @Security ApiKeyAuth
// @Router /network/site/exports/task/{taskId}/cancel [post]
func CancelSiteExportTaskHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	if !siteExportManager.CancelTask(id) {
		common.Error(w, r, http.StatusNotFound, http.StatusNotFound, "task not found or not cancelable")
		return
	}
	common.Success(w, r, "success")
}

// PreviewSiteExportHandler godoc
// @Summary Preview site export results
// @Description Evaluates the export rule and returns matched entries (no pagination).
// @Tags network/site
// @Accept json
// @Produce json
// @Param request body models.SiteExportPreviewRequest true "Preview Request"
// @Success 200 {array} models.SitePoolEntry
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/site/exports/preview [post]
func PreviewSiteExportHandler(w http.ResponseWriter, r *http.Request) {

	var req models.SiteExportPreviewRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	res, err := sitePoolService.PreviewExport(r.Context(), &req)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// DownloadSiteExportHandler godoc
// @Summary Download site export result
// @Tags network/site
// @Produce octet-stream
// @Param taskId path string true "Task ID"
// @Success 200 {file} file
// @Router /network/site/exports/download/{taskId} [get]
func DownloadSiteExportHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "taskId")
	task := siteExportManager.GetTask(id)
	if task == nil || task.Status != "Success" {
		http.Error(w, "file not ready", http.StatusNotFound)
		return
	}
	tempFileName := fmt.Sprintf("site_export_%s.%s", id, task.Format)
	f, err := common.TempDir.Open(filepath.Join("temp", tempFileName))
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", tempFileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, f)
}

// Site Sync Policies

// ScanSiteSyncPoliciesHandler godoc
// @Summary Scan all site sync policies
// @Tags network/site
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name"
// @Success 200 {object} common.CursorResponse{items=[]models.SiteSyncPolicy}
// @Router /network/site/sync [get]
func ScanSiteSyncPoliciesHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := sitePoolService.ScanSyncPolicies(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateSiteSyncPolicyHandler godoc
// @Summary Create a site sync policy
// @Tags network/site
// @Accept json
// @Produce json
// @Param policy body models.SiteSyncPolicy true "Site Sync Policy"
// @Success 200 {object} models.SiteSyncPolicy
// @Router /network/site/sync [post]
func CreateSiteSyncPolicyHandler(w http.ResponseWriter, r *http.Request) {
	var policy models.SiteSyncPolicy
	if err := render.Bind(r, &policy); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := sitePoolService.CreateSyncPolicy(r.Context(), &policy); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, policy)
}

// UpdateSiteSyncPolicyHandler godoc
// @Summary Update a site sync policy
// @Tags network/site
// @Accept json
// @Produce json
// @Param id path string true "Policy ID"
// @Param policy body models.SiteSyncPolicy true "Site Sync Policy"
// @Success 200 {object} models.SiteSyncPolicy
// @Router /network/site/sync/{id} [put]
func UpdateSiteSyncPolicyHandler(w http.ResponseWriter, r *http.Request) {
	var policy models.SiteSyncPolicy
	if err := render.Bind(r, &policy); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	policy.ID = chi.URLParam(r, "id")
	if err := sitePoolService.UpdateSyncPolicy(r.Context(), &policy); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, policy)
}

// DeleteSiteSyncPolicyHandler godoc
// @Summary Delete a site sync policy
// @Tags network/site
// @Produce json
// @Param id path string true "Policy ID"
// @Success 200 {string} string "success"
// @Router /network/site/sync/{id} [delete]
func DeleteSiteSyncPolicyHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := sitePoolService.DeleteSyncPolicy(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// TriggerSiteSyncHandler godoc
// @Summary Trigger a site sync policy execution
// @Tags network/site
// @Produce json
// @Param id path string true "Policy ID"
// @Success 200 {string} string "success"
// @Router /network/site/sync/{id}/trigger [post]
func TriggerSiteSyncHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := sitePoolService.Sync(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "sync started")
}

// UpdateSiteGroupHandler godoc
// @Summary Update a site group
// @Tags network/site
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param group body models.SiteGroup true "Site Group"
// @Success 200 {object} models.SiteGroup
// @Router /network/site/pools/{id} [put]
func UpdateSiteGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var group models.SiteGroup
	if err := render.Bind(r, &group); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	group.ID = id
	if err := sitePoolService.UpdateGroup(r.Context(), &group); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, group)
}

// SiteRouter registers the site routes
func SiteRouter(r chi.Router) {
	r.With(middlewares.RequirePermission("list", "network/site")).Get("/api/v1/network/site/pools", ScanSiteGroupsHandler)
	r.With(middlewares.RequirePermission("create", "network/site")).Post("/api/v1/network/site/pools", CreateSiteGroupHandler)
	r.With(middlewares.RequirePermission("update", "network/site")).Put("/api/v1/network/site/pools/{id}", UpdateSiteGroupHandler)
	r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/api/v1/network/site/pools/{id}", DeleteSiteGroupHandler)
	r.With(middlewares.RequirePermission("get", "network/site")).Get("/api/v1/network/site/pools/{id}/preview", PreviewSitePoolHandler)
	r.With(middlewares.RequirePermission("update", "network/site")).Post("/api/v1/network/site/pools/{id}/entries", ManageSitePoolEntryHandler)
	r.With(middlewares.RequirePermission("update", "network/site")).Delete("/api/v1/network/site/pools/{id}/entries", DeleteSitePoolEntryHandler)

	r.With(middlewares.RequirePermission("execute", "network/site")).Post("/api/v1/network/site/analysis/hit-test", SiteHitTestHandler)

	r.With(middlewares.RequirePermission("list", "network/site")).Get("/api/v1/network/site/exports", ScanSiteExportsHandler)
	r.With(middlewares.RequirePermission("list", "network/site")).Get("/api/v1/network/site/exports/tasks", ScanSiteExportTasksHandler)
	r.With(middlewares.RequirePermission("create", "network/site")).Post("/api/v1/network/site/exports", CreateSiteExportHandler)
	r.With(middlewares.RequirePermission("update", "network/site")).Put("/api/v1/network/site/exports/{id}", UpdateSiteExportHandler)
	r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/api/v1/network/site/exports/{id}", DeleteSiteExportHandler)
	r.With(middlewares.RequirePermission("execute", "network/site")).Post("/api/v1/network/site/exports/{id}/trigger", TriggerSiteExportHandler)
	r.With(middlewares.RequirePermission("get", "network/site")).Get("/api/v1/network/site/exports/task/{taskId}", SiteExportTaskStatusHandler)
	r.With(middlewares.RequirePermission("execute", "network/site")).Post("/api/v1/network/site/exports/task/{taskId}/cancel", CancelSiteExportTaskHandler)
	r.With(middlewares.RequirePermission("get", "network/site")).Get("/api/v1/network/site/exports/download/{taskId}", DownloadSiteExportHandler)
	r.With(middlewares.RequirePermission("execute", "network/site")).Post("/api/v1/network/site/exports/preview", PreviewSiteExportHandler)

	r.With(middlewares.RequirePermission("list", "network/site")).Get("/api/v1/network/site/sync", ScanSiteSyncPoliciesHandler)
	r.With(middlewares.RequirePermission("create", "network/site")).Post("/api/v1/network/site/sync", CreateSiteSyncPolicyHandler)
	r.With(middlewares.RequirePermission("update", "network/site")).Put("/api/v1/network/site/sync/{id}", UpdateSiteSyncPolicyHandler)
	r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/api/v1/network/site/sync/{id}", DeleteSiteSyncPolicyHandler)
	r.With(middlewares.RequirePermission("execute", "network/site")).Post("/api/v1/network/site/sync/{id}/trigger", TriggerSiteSyncHandler)
}
