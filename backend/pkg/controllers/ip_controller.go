package controllers

import (
	"fmt"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/controllers/middlewares"
	"homelab/pkg/models"
	ipservice "homelab/pkg/services/ip"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

var ipPoolService *ipservice.IPPoolService
var analysisEngine *ipservice.AnalysisEngine
var exportManager *ipservice.ExportManager

func InitIPControllers(service *ipservice.IPPoolService, engine *ipservice.AnalysisEngine, em *ipservice.ExportManager) {
	ipPoolService = service
	analysisEngine = engine
	exportManager = em
}

// ScanGroupsHandler godoc
// @Summary Scan all IP groups
// @Tags network/ip
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name"
// @Success 200 {object} common.CursorResponse{items=[]models.IPPool}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/pools [get]
func ScanGroupsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := ipPoolService.ScanGroups(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateGroupHandler godoc
// @Summary Create an IP group
// @Tags network/ip
// @Accept json
// @Produce json
// @Param group body models.IPPool true "IP Group"
// @Success 200 {object} models.IPPool
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/pools [post]
func CreateGroupHandler(w http.ResponseWriter, r *http.Request) {
	var group models.IPPool
	if err := render.Bind(r, &group); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := ipPoolService.CreateGroup(r.Context(), &group); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, group)
}

// UpdateGroupHandler godoc
// @Summary Update an IP group
// @Tags network/ip
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param group body models.IPPool true "IP Group"
// @Success 200 {object} models.IPPool
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Group Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/pools/{id} [put]
func UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/ip/"+id) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/ip") {
		HandleError(w, r, fmt.Errorf("%w: network/ip/%s", commonauth.ErrPermissionDenied, id))
		return
	}
	var group models.IPPool
	if err := render.Bind(r, &group); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	group.ID = id
	if err := ipPoolService.UpdateGroup(r.Context(), &group); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, group)
}

// PreviewPoolHandler godoc
// @Summary Preview IP pool data (cursor-based)
// @Tags network/ip
// @Produce json
// @Param id path string true "Group ID"
// @Param cursor query string false "Byte offset cursor"
// @Param limit query int false "Number of entries to return"
// @Param search query string false "Search prefix or tag"
// @Success 200 {object} models.IPPoolPreviewResponse
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Group Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/pools/{id}/preview [get]
func PreviewPoolHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/ip/"+id) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/ip") {
		HandleError(w, r, fmt.Errorf("%w: network/ip/%s", commonauth.ErrPermissionDenied, id))
		return
	}
	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 {
		limit = 100
	}
	search := r.URL.Query().Get("search")

	res, err := ipPoolService.PreviewPool(r.Context(), id, cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// DeleteGroupHandler godoc
// @Summary Delete an IP group
// @Tags network/ip
// @Produce json
// @Param id path string true "Group ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Group Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/pools/{id} [delete]
func DeleteGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/ip/"+id) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/ip") {
		HandleError(w, r, fmt.Errorf("%w: network/ip/%s", commonauth.ErrPermissionDenied, id))
		return
	}
	if err := ipPoolService.DeleteGroup(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ScanExportsHandler godoc
// @Summary Scan all IP exports
// @Tags network/ip
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name"
// @Success 200 {object} common.CursorResponse{items=[]models.IPExport}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/exports [get]
func ScanExportsHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := ipPoolService.ScanExports(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateExportHandler godoc
// @Summary Create an IP export
// @Tags network/ip
// @Accept json
// @Produce json
// @Param export body models.IPExport true "IP Export"
// @Success 200 {object} models.IPExport
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/exports [post]
func CreateExportHandler(w http.ResponseWriter, r *http.Request) {
	var export models.IPExport
	if err := render.Bind(r, &export); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := ipPoolService.CreateExport(r.Context(), &export); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, export)
}

// UpdateExportHandler godoc
// @Summary Update an IP export
// @Tags network/ip
// @Accept json
// @Produce json
// @Param id path string true "Export ID"
// @Param export body models.IPExport true "IP Export"
// @Success 200 {object} models.IPExport
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Export Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/exports/{id} [put]
func UpdateExportHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var export models.IPExport
	if err := render.Bind(r, &export); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	export.ID = id
	if err := ipPoolService.UpdateExport(r.Context(), &export); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, export)
}

// DeleteExportHandler godoc
// @Summary Delete an IP export
// @Tags network/ip
// @Produce json
// @Param id path string true "Export ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Export Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/exports/{id} [delete]
func DeleteExportHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := ipPoolService.DeleteExport(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// HitTestHandler godoc
// @Summary Perform IP hit test
// @Tags network/ip
// @Accept json
// @Produce json
// @Param request body models.IPHitTestRequest true "Hit test request"
// @Success 200 {object} models.IPAnalysisResult
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/analysis/hit-test [post]
func HitTestHandler(w http.ResponseWriter, r *http.Request) {
	var req models.IPHitTestRequest
	if err := render.DecodeJSON(r.Body, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	res, err := analysisEngine.HitTest(r.Context(), req.IP, req.GroupIDs)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// IPInfoHandler godoc
// @Summary Get IP intelligence info
// @Tags network/ip
// @Produce json
// @Param ip query string true "IP address"
// @Success 200 {object} models.IPInfoResponse
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/analysis/info [get]
func IPInfoHandler(w http.ResponseWriter, r *http.Request) {
	ipStr := r.URL.Query().Get("ip")
	res, err := analysisEngine.Info(r.Context(), ipStr)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// ScanExportTasksHandler godoc
// @Summary Scan all IP export tasks
// @Tags network/ip
// @Produce json
// @Success 200 {array} ipservice.ExportTask
// @Security ApiKeyAuth
// @Router /network/ip/exports/tasks [get]
func ScanExportTasksHandler(w http.ResponseWriter, r *http.Request) {
	tasks := exportManager.ScanTasks()
	common.Success(w, r, tasks)
}

// TriggerExportHandler godoc
// @Summary Trigger dynamic export
// @Tags network/ip
// @Produce json
// @Param id path string true "Export ID"
// @Param format query string false "Format: text, json, yaml"
// @Success 200 {object} models.IPExportTriggerResponse
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Export Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/exports/{id}/trigger [post]
func TriggerExportHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "text"
	}
	taskID, err := exportManager.TriggerExport(r.Context(), id, format)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, &models.IPExportTriggerResponse{TaskID: taskID})
}

// ExportTaskStatusHandler godoc
// @Summary Get export task status
// @Tags network/ip
// @Produce json
// @Param taskId path string true "Task ID"
// @Success 200 {object} ipservice.ExportTask
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Task Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/exports/task/{taskId} [get]
func ExportTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	task := exportManager.GetTask(taskId)
	if task == nil {
		common.Error(w, r, http.StatusNotFound, http.StatusNotFound, "task not found")
		return
	}
	common.Success(w, r, task)
}

// CancelExportTaskHandler godoc
// @Summary Cancel an IP export task
// @Tags network/ip
// @Produce json
// @Param taskId path string true "Task ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Task Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/exports/task/{taskId}/cancel [post]
func CancelExportTaskHandler(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	if !exportManager.CancelTask(taskId) {
		common.Error(w, r, http.StatusNotFound, http.StatusNotFound, "task not found or not cancelable")
		return
	}
	common.Success(w, r, "success")
}

// PreviewExportHandler godoc
// @Summary Preview IP export expression
// @Description Evaluates the export rule and returns matched entries (no pagination).
// @Tags network/ip
// @Accept json
// @Produce json
// @Param request body models.IPExportPreviewRequest true "Preview Request"
// @Success 200 {array} models.IPPoolEntry
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/exports/preview [post]
func PreviewExportHandler(w http.ResponseWriter, r *http.Request) {
	var req models.IPExportPreviewRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	res, err := ipPoolService.PreviewExport(r.Context(), &req)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, res)
}

// DownloadExportHandler godoc
// @Summary Download export result
// @Tags network/ip
// @Produce octet-stream
// @Param taskId path string true "Task ID"
// @Success 200 {file} file
// @Failure 404 {object} common.Response "File Not Found"
// @Router /network/ip/exports/download/{taskId} [get]
func DownloadExportHandler(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	task := exportManager.GetTask(taskId)
	if task == nil || task.Status != "Success" {
		http.Error(w, "file not ready or not found", http.StatusNotFound)
		return
	}

	tempFileName := fmt.Sprintf("export_%s.%s", taskId, task.Format)
	tempPath := filepath.Join("temp", tempFileName)

	f, err := common.TempDir.Open(tempPath)
	if err != nil {
		http.Error(w, "file not found", http.StatusNotFound)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", tempFileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, f)
}

// ManagePoolEntryHandler godoc
// @Summary Add or update a tag for a CIDR in IP pool
// @Description If oldTag is empty, adds newTag. If both provided, renames oldTag to newTag.
// @Tags network/ip
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param entry body models.IPPoolEntryRequest true "IP/CIDR Tag Entry"
// @Success 200 {string} string "success"
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Group Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/pools/{id}/entries [post]
func ManagePoolEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/ip/"+id) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/ip") {
		HandleError(w, r, fmt.Errorf("%w: network/ip/%s", commonauth.ErrPermissionDenied, id))
		return
	}
	var req models.IPPoolEntryRequest
	if err := render.Bind(r, &req); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	mode := "add"
	if len(req.OldTags) > 0 || len(req.NewTags) > 0 {
		mode = "update"
	}
	// 如果是全新添加 CIDR (Old为空，只有New)，ManagePoolEntry 内部已处理为合并模式
	if err := ipPoolService.ManagePoolEntry(r.Context(), id, &req, mode); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// DeletePoolEntryHandler godoc
// @Summary Delete an entry or a specific tag from IP pool
// @Tags network/ip
// @Produce json
// @Param id path string true "Group ID"
// @Param cidr query string true "CIDR or IP to delete"
// @Param tag query string false "Specific tag to delete (if omitted, deletes entire CIDR if no internal tags present)"
// @Success 200 {string} string "success"
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Group Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/pools/{id}/entries [delete]
func DeletePoolEntryHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/ip/"+id) && !commonauth.PermissionsFromContext(r.Context()).IsAllowed("network/ip") {
		HandleError(w, r, fmt.Errorf("%w: network/ip/%s", commonauth.ErrPermissionDenied, id))
		return
	}
	cidr := r.URL.Query().Get("cidr")
	tagsStr := r.URL.Query().Get("tags")
	var tags []string
	if tagsStr != "" {
		tags = strings.Split(tagsStr, ",")
	}
	if cidr == "" {
		common.BadRequestError(w, r, http.StatusBadRequest, "cidr is required")
		return
	}
	req := models.IPPoolEntryRequest{CIDR: cidr, OldTags: tags}
	if err := ipPoolService.ManagePoolEntry(r.Context(), id, &req, "delete"); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ScanSyncPoliciesHandler godoc
// @Summary Scan all IP sync policies
// @Tags network/ip
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name"
// @Success 200 {object} common.CursorResponse{items=[]models.IPSyncPolicy}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/sync [get]
func ScanSyncPoliciesHandler(w http.ResponseWriter, r *http.Request) {
	cursor, limit := getCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := ipPoolService.ScanSyncPolicies(r.Context(), cursor, limit, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, res)
}

// CreateSyncPolicyHandler godoc
// @Summary Create an IP sync policy
// @Tags network/ip
// @Accept json
// @Produce json
// @Param policy body models.IPSyncPolicy true "IP Sync Policy"
// @Success 200 {object} models.IPSyncPolicy
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/sync [post]
func CreateSyncPolicyHandler(w http.ResponseWriter, r *http.Request) {
	var policy models.IPSyncPolicy
	if err := render.Bind(r, &policy); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := ipPoolService.CreateSyncPolicy(r.Context(), &policy); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, policy)
}

// UpdateSyncPolicyHandler godoc
// @Summary Update an IP sync policy
// @Tags network/ip
// @Accept json
// @Produce json
// @Param id path string true "Policy ID"
// @Param policy body models.IPSyncPolicy true "IP Sync Policy"
// @Success 200 {object} models.IPSyncPolicy
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/sync/{id} [put]
func UpdateSyncPolicyHandler(w http.ResponseWriter, r *http.Request) {
	var policy models.IPSyncPolicy
	if err := render.Bind(r, &policy); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	policy.ID = chi.URLParam(r, "id")
	if err := ipPoolService.UpdateSyncPolicy(r.Context(), &policy); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, policy)
}

// DeleteSyncPolicyHandler godoc
// @Summary Delete an IP sync policy
// @Tags network/ip
// @Produce json
// @Param id path string true "Policy ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Policy Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/sync/{id} [delete]
func DeleteSyncPolicyHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := ipPoolService.DeleteSyncPolicy(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// TriggerSyncHandler godoc
// @Summary Trigger manual sync
// @Tags network/ip
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {string} string "sync started"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Policy Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/sync/{id}/trigger [post]
func TriggerSyncHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := ipPoolService.Sync(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "sync started")
}

// IPRouter registers the IP routes
func IPRouter(r chi.Router) {
	r.With(middlewares.RequirePermission("list", "network/ip")).Get("/api/v1/network/ip/pools", ScanGroupsHandler)
	r.With(middlewares.RequirePermission("create", "network/ip")).Post("/api/v1/network/ip/pools", CreateGroupHandler)
	r.With(middlewares.RequirePermission("update", "network/ip")).Put("/api/v1/network/ip/pools/{id}", UpdateGroupHandler)
	r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/api/v1/network/ip/pools/{id}", DeleteGroupHandler)
	r.With(middlewares.RequirePermission("get", "network/ip")).Get("/api/v1/network/ip/pools/{id}/preview", PreviewPoolHandler)
	r.With(middlewares.RequirePermission("update", "network/ip")).Post("/api/v1/network/ip/pools/{id}/entries", ManagePoolEntryHandler)
	r.With(middlewares.RequirePermission("update", "network/ip")).Delete("/api/v1/network/ip/pools/{id}/entries", DeletePoolEntryHandler)

	r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/api/v1/network/ip/analysis/hit-test", HitTestHandler)
	r.With(middlewares.RequirePermission("get", "network/ip")).Get("/api/v1/network/ip/analysis/info", IPInfoHandler)

	r.With(middlewares.RequirePermission("list", "network/ip")).Get("/api/v1/network/ip/exports", ScanExportsHandler)
	r.With(middlewares.RequirePermission("list", "network/ip")).Get("/api/v1/network/ip/exports/tasks", ScanExportTasksHandler)
	r.With(middlewares.RequirePermission("create", "network/ip")).Post("/api/v1/network/ip/exports", CreateExportHandler)
	r.With(middlewares.RequirePermission("update", "network/ip")).Put("/api/v1/network/ip/exports/{id}", UpdateExportHandler)
	r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/api/v1/network/ip/exports/{id}", DeleteExportHandler)
	r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/api/v1/network/ip/exports/{id}/trigger", TriggerExportHandler)
	r.With(middlewares.RequirePermission("get", "network/ip")).Get("/api/v1/network/ip/exports/task/{taskId}", ExportTaskStatusHandler)
	r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/api/v1/network/ip/exports/task/{taskId}/cancel", CancelExportTaskHandler)
	r.With(middlewares.RequirePermission("get", "network/ip")).Get("/api/v1/network/ip/exports/download/{taskId}", DownloadExportHandler)
	r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/api/v1/network/ip/exports/preview", PreviewExportHandler)

	r.With(middlewares.RequirePermission("list", "network/ip")).Get("/api/v1/network/ip/sync", ScanSyncPoliciesHandler)
	r.With(middlewares.RequirePermission("create", "network/ip")).Post("/api/v1/network/ip/sync", CreateSyncPolicyHandler)
	r.With(middlewares.RequirePermission("update", "network/ip")).Put("/api/v1/network/ip/sync/{id}", UpdateSyncPolicyHandler)
	r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/api/v1/network/ip/sync/{id}", DeleteSyncPolicyHandler)
	r.With(middlewares.RequirePermission("execute", "network/ip")).Post("/api/v1/network/ip/sync/{id}/trigger", TriggerSyncHandler)
}
