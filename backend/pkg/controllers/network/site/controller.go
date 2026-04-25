package site

import (
	"fmt"
	apiv1 "homelab/pkg/apis/network/site/v1"
	"homelab/pkg/common"
	controllercommon "homelab/pkg/controllers"
	"io"
	"net/http"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	cursor, limit := controllercommon.GetCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := deps.PoolService.ScanGroups(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapGroups(res))
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	var group apiv1.Group
	if err := render.Bind(r, &group); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	model := toModelGroup(group)
	if err := deps.PoolService.CreateGroup(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIGroup(model))
}

// DeleteSiteGroupHandler godoc
// @Summary Delete a site group
// @Tags network/site
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {string} string "success"
// @Router /network/site/pools/{id} [delete]
func DeleteSiteGroupHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := controllercommon.RequireScopedPermission(r.Context(), controllercommon.NetworkSiteResourceBase, id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	if err := deps.PoolService.DeleteGroup(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := controllercommon.RequireScopedPermission(r.Context(), controllercommon.NetworkSiteResourceBase, id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	cursor := r.URL.Query().Get("cursor")
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 {
		limit = 100
	}
	search := r.URL.Query().Get("search")
	res, err := deps.PoolService.PreviewPool(r.Context(), id, cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIPoolPreview(res))
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := controllercommon.RequireScopedPermission(r.Context(), controllercommon.NetworkSiteResourceBase, id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	var req apiv1.PoolEntryRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	modelReq := toModelPoolEntryRequest(req)
	if err := deps.PoolService.ManagePoolEntry(r.Context(), id, &modelReq, "add"); err != nil {
		controllercommon.HandleError(w, r, err)
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := controllercommon.RequireScopedPermission(r.Context(), controllercommon.NetworkSiteResourceBase, id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	value := r.URL.Query().Get("value")

	t, _ := strconv.Atoi(r.URL.Query().Get("type"))
	req := apiv1.PoolEntryRequest{Type: uint8(t), Value: value}
	modelReq := toModelPoolEntryRequest(req)
	if err := deps.PoolService.ManagePoolEntry(r.Context(), id, &modelReq, "delete"); err != nil {
		controllercommon.HandleError(w, r, err)
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	var req struct {
		Domain   string   `json:"domain"`
		GroupIDs []string `json:"groupIds"`
	}
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	res, err := deps.Analysis.HitTest(r.Context(), req.Domain, req.GroupIDs)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIAnalysisResult(res))
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	cursor, limit := controllercommon.GetCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := deps.PoolService.ScanExports(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapExports(res))
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	var export apiv1.Export
	if err := render.Bind(r, &export); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	model := toModelExport(export)
	if err := deps.PoolService.CreateExport(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIExport(model))
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var export apiv1.Export
	if err := render.Bind(r, &export); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	export.ID = id
	model := toModelExport(export)
	if err := deps.PoolService.UpdateExport(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIExport(model))
}

// DeleteSiteExportHandler godoc
// @Summary Delete a site export
// @Tags network/site
// @Produce json
// @Param id path string true "Export ID"
// @Success 200 {string} string "success"
// @Router /network/site/exports/{id} [delete]
func DeleteSiteExportHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := deps.PoolService.DeleteExport(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	tasks := deps.Exports.ScanTasks()

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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "text"
	}
	taskID, err := deps.Exports.TriggerExport(r.Context(), id, format)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, &apiv1.ExportTriggerResponse{TaskID: taskID})
}

// SiteExportTaskStatusHandler godoc
// @Summary Get site export task status
// @Tags network/site
// @Produce json
// @Param taskId path string true "Task ID"
// @Success 200 {object} siteservice.ExportTask
// @Router /network/site/exports/task/{taskId} [get]
func SiteExportTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "taskId")
	task := deps.Exports.GetTask(id)
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "taskId")
	if !deps.Exports.CancelTask(id) {
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}

	var req apiv1.ExportPreviewRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	modelReq := toModelExportPreviewRequest(req)
	res, err := deps.PoolService.PreviewExport(r.Context(), &modelReq)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIPoolEntries(res))
}

// DownloadSiteExportHandler godoc
// @Summary Download site export result
// @Tags network/site
// @Produce octet-stream
// @Param taskId path string true "Task ID"
// @Success 200 {file} file
// @Router /network/site/exports/download/{taskId} [get]
func DownloadSiteExportHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "taskId")
	task := deps.Exports.GetTask(id)
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	cursor, limit := controllercommon.GetCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := deps.PoolService.ScanSyncPolicies(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapSyncPolicies(res))
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	var policy apiv1.SyncPolicy
	if err := render.Bind(r, &policy); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	model := toModelSyncPolicy(policy)
	if err := deps.PoolService.CreateSyncPolicy(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPISyncPolicy(model))
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	var policy apiv1.SyncPolicy
	if err := render.Bind(r, &policy); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	policy.ID = chi.URLParam(r, "id")
	model := toModelSyncPolicy(policy)
	if err := deps.PoolService.UpdateSyncPolicy(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPISyncPolicy(model))
}

// DeleteSiteSyncPolicyHandler godoc
// @Summary Delete a site sync policy
// @Tags network/site
// @Produce json
// @Param id path string true "Policy ID"
// @Success 200 {string} string "success"
// @Router /network/site/sync/{id} [delete]
func DeleteSiteSyncPolicyHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := deps.PoolService.DeleteSyncPolicy(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := deps.PoolService.Sync(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
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
	deps, ok := controllercommon.SiteDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var group apiv1.Group
	if err := render.Bind(r, &group); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	group.ID = id
	model := toModelGroup(group)
	if err := deps.PoolService.UpdateGroup(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIGroup(model))
}
