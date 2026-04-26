package ip

import (
	"fmt"
	apiv1 "homelab/pkg/apis/network/ip/v1"
	"homelab/pkg/common"
	controllercommon "homelab/pkg/controllers"
	"io"
	"net/http"
	"path/filepath"
	"strings"
)

// ScanGroupsHandler godoc
// @Summary Scan all IP groups
// @Tags network/ip
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search by name"
// @Success 200 {object} common.CursorResponse{items=[]apiv1.Pool}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/pools [get]
func ScanGroupsHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	cursor, limit, search := controllercommon.GetSearchCursorParams(r)
	res, err := deps.PoolService.ScanGroups(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapPools(res))
}

// CreateGroupHandler godoc
// @Summary Create an IP group
// @Tags network/ip
// @Accept json
// @Produce json
// @Param group body apiv1.Pool true "IP Group"
// @Success 200 {object} apiv1.Pool
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/pools [post]
func CreateGroupHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	group, ok := controllercommon.BindRequest[apiv1.Pool](w, r)
	if !ok {
		return
	}
	model := toModelPool(group)
	if err := deps.PoolService.CreateGroup(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIPool(model))
}

// UpdateGroupHandler godoc
// @Summary Update an IP group
// @Tags network/ip
// @Accept json
// @Produce json
// @Param id path string true "Group ID"
// @Param group body apiv1.Pool true "IP Group"
// @Success 200 {object} apiv1.Pool
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Group Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/pools/{id} [put]
func UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	id := controllercommon.PathID(r, "id")
	group, ok := controllercommon.BindRequest[apiv1.Pool](w, r)
	if !ok {
		return
	}
	group.ID = id
	model := toModelPool(group)
	if err := deps.PoolService.UpdateGroup(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIPool(model))
}

// PreviewPoolHandler godoc
// @Summary Preview IP pool data (cursor-based)
// @Tags network/ip
// @Produce json
// @Param id path string true "Group ID"
// @Param cursor query string false "Byte offset cursor"
// @Param limit query int false "Number of entries to return"
// @Param search query string false "Search prefix or tag"
// @Success 200 {object} apiv1.PoolPreviewResponse
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Group Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/pools/{id}/preview [get]
func PreviewPoolHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	id := controllercommon.PathID(r, "id")
	cursor, limit, search := controllercommon.GetSearchCursorParamsWithDefault(r, 100)

	res, err := deps.PoolService.PreviewPool(r.Context(), id, cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIPoolPreview(res))
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
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	id := controllercommon.PathID(r, "id")
	if err := deps.PoolService.DeleteGroup(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
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
// @Success 200 {object} common.CursorResponse{items=[]apiv1.Export}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/exports [get]
func ScanExportsHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	cursor, limit, search := controllercommon.GetSearchCursorParams(r)
	res, err := deps.PoolService.ScanExports(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapExports(res))
}

// CreateExportHandler godoc
// @Summary Create an IP export
// @Tags network/ip
// @Accept json
// @Produce json
// @Param export body apiv1.Export true "IP Export"
// @Success 200 {object} apiv1.Export
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/exports [post]
func CreateExportHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	export, ok := controllercommon.BindRequest[apiv1.Export](w, r)
	if !ok {
		return
	}
	model := toModelExport(export)
	if err := deps.PoolService.CreateExport(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIExport(model))
}

// UpdateExportHandler godoc
// @Summary Update an IP export
// @Tags network/ip
// @Accept json
// @Produce json
// @Param id path string true "Export ID"
// @Param export body apiv1.Export true "IP Export"
// @Success 200 {object} apiv1.Export
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Export Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/exports/{id} [put]
func UpdateExportHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	id := controllercommon.PathID(r, "id")
	export, ok := controllercommon.BindRequest[apiv1.Export](w, r)
	if !ok {
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
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	id := controllercommon.PathID(r, "id")
	if err := deps.PoolService.DeleteExport(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// HitTestHandler godoc
// @Summary Perform IP hit test
// @Tags network/ip
// @Accept json
// @Produce json
// @Param request body apiv1.HitTestRequest true "Hit test request"
// @Success 200 {object} apiv1.AnalysisResult
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/analysis/hit-test [post]
func HitTestHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	req, ok := controllercommon.BindRequest[apiv1.HitTestRequest](w, r)
	if !ok {
		return
	}
	res, err := deps.Analysis.HitTest(r.Context(), req.IP, req.GroupIDs)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIAnalysisResult(res))
}

// IPInfoHandler godoc
// @Summary Get IP intelligence info
// @Tags network/ip
// @Produce json
// @Param ip query string true "IP address"
// @Success 200 {object} apiv1.IPInfoResponse
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/analysis/info [get]
func IPInfoHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	ipStr := r.URL.Query().Get("ip")
	res, err := deps.Analysis.Info(r.Context(), ipStr)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPIIPInfo(res))
}

// ScanExportTasksHandler godoc
// @Summary Scan all IP export tasks
// @Tags network/ip
// @Produce json
// @Success 200 {array} apiv1.ExportTask
// @Security ApiKeyAuth
// @Router /network/ip/exports/tasks [get]
func ScanExportTasksHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	tasks := deps.Exports.ScanTasks()
	common.Success(w, r, toAPIExportTasks(tasks))
}

// TriggerIPExportHandler godoc
// @Summary Trigger dynamic export
// @Tags network/ip
// @Produce json
// @Param id path string true "Export ID"
// @Param format query string false "Format: text, json, yaml"
// @Success 200 {object} apiv1.ExportTriggerResponse
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Export Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/exports/{id}/trigger [post]
func TriggerIPExportHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	id := controllercommon.PathID(r, "id")
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

// ExportTaskStatusHandler godoc
// @Summary Get export task status
// @Tags network/ip
// @Produce json
// @Param taskId path string true "Task ID"
// @Success 200 {object} apiv1.ExportTask
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Task Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/exports/task/{taskId} [get]
func ExportTaskStatusHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	taskId := controllercommon.PathID(r, "taskId")
	task := deps.Exports.GetTask(taskId)
	if task == nil {
		common.Error(w, r, http.StatusNotFound, http.StatusNotFound, "task not found")
		return
	}
	common.Success(w, r, toAPIExportTask(*task))
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
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	taskId := controllercommon.PathID(r, "taskId")
	if !deps.Exports.CancelTask(taskId) {
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
// @Param request body apiv1.ExportPreviewRequest true "Preview Request"
// @Success 200 {array} apiv1.PoolEntry
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/exports/preview [post]
func PreviewExportHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	req, ok := controllercommon.BindRequest[apiv1.ExportPreviewRequest](w, r)
	if !ok {
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

// DownloadExportHandler godoc
// @Summary Download export result
// @Tags network/ip
// @Produce octet-stream
// @Param taskId path string true "Task ID"
// @Success 200 {file} file
// @Failure 404 {object} common.Response "File Not Found"
// @Router /network/ip/exports/download/{taskId} [get]
func DownloadExportHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	taskId := controllercommon.PathID(r, "taskId")
	task := deps.Exports.GetTask(taskId)
	if task == nil || task.Status != "Success" {
		http.Error(w, "file not ready or not found", http.StatusNotFound)
		return
	}

	tempFileName := fmt.Sprintf("export_%s.%s", taskId, task.Format)
	tempPath := filepath.Join("temp", tempFileName)

	f, err := deps.TempFS.Open(tempPath)
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
// @Param entry body apiv1.PoolEntryRequest true "IP/CIDR Tag Entry"
// @Success 200 {string} string "success"
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Group Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/pools/{id}/entries [post]
func ManagePoolEntryHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	id := controllercommon.PathID(r, "id")
	req, ok := controllercommon.BindRequest[apiv1.PoolEntryRequest](w, r)
	if !ok {
		return
	}
	mode := "add"
	if len(req.OldTags) > 0 || len(req.NewTags) > 0 {
		mode = "update"
	}
	// 如果是全新添加 CIDR (Old为空，只有New)，ManagePoolEntry 内部已处理为合并模式
	modelReq := toModelPoolEntryRequest(req)
	if err := deps.PoolService.ManagePoolEntry(r.Context(), id, &modelReq, mode); err != nil {
		controllercommon.HandleError(w, r, err)
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
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	id := controllercommon.PathID(r, "id")
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
	req := apiv1.PoolEntryRequest{CIDR: cidr, OldTags: tags}
	modelReq := toModelPoolEntryRequest(req)
	if err := deps.PoolService.ManagePoolEntry(r.Context(), id, &modelReq, "delete"); err != nil {
		controllercommon.HandleError(w, r, err)
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
// @Success 200 {object} common.CursorResponse{items=[]apiv1.SyncPolicy}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/sync [get]
func ScanSyncPoliciesHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	cursor, limit, search := controllercommon.GetSearchCursorParams(r)
	res, err := deps.PoolService.ScanSyncPolicies(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapSyncPolicies(res))
}

// CreateSyncPolicyHandler godoc
// @Summary Create an IP sync policy
// @Tags network/ip
// @Accept json
// @Produce json
// @Param policy body apiv1.SyncPolicy true "IP Sync Policy"
// @Success 200 {object} apiv1.SyncPolicy
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/sync [post]
func CreateSyncPolicyHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	policy, ok := controllercommon.BindRequest[apiv1.SyncPolicy](w, r)
	if !ok {
		return
	}
	model := toModelSyncPolicy(policy)
	if err := deps.PoolService.CreateSyncPolicy(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPISyncPolicy(model))
}

// UpdateSyncPolicyHandler godoc
// @Summary Update an IP sync policy
// @Tags network/ip
// @Accept json
// @Produce json
// @Param id path string true "Policy ID"
// @Param policy body apiv1.SyncPolicy true "IP Sync Policy"
// @Success 200 {object} apiv1.SyncPolicy
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/sync/{id} [put]
func UpdateSyncPolicyHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	policy, ok := controllercommon.BindRequest[apiv1.SyncPolicy](w, r)
	if !ok {
		return
	}
	policy.ID = controllercommon.PathID(r, "id")
	model := toModelSyncPolicy(policy)
	if err := deps.PoolService.UpdateSyncPolicy(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPISyncPolicy(model))
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
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	id := controllercommon.PathID(r, "id")
	if err := deps.PoolService.DeleteSyncPolicy(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
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
	deps, ok := depsFromRequest(w, r)
	if !ok {
		return
	}
	id := controllercommon.PathID(r, "id")
	if err := deps.PoolService.Sync(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, "sync started")
}
