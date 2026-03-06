package controllers

import (
	"fmt"
	"io"
	"homelab/pkg/common"
	"homelab/pkg/controllers/middlewares"
	"homelab/pkg/models"
	ipservice "homelab/pkg/services/ip"
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

// ListGroupsHandler godoc
// @Summary List all IP groups
// @Tags network/ip
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name"
// @Success 200 {object} common.PaginatedResponse{items=[]models.IPGroup}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/pools [get]
func ListGroupsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	items, total, err := ipPoolService.ListGroups(r.Context(), page, pageSize, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.PaginatedSuccess(w, r, items, total, page, pageSize)
}

// CreateGroupHandler godoc
// @Summary Create an IP group
// @Tags network/ip
// @Accept json
// @Produce json
// @Param group body models.IPGroup true "IP Group"
// @Success 200 {object} models.IPGroup
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/ip/pools [post]
func CreateGroupHandler(w http.ResponseWriter, r *http.Request) {
	var group models.IPGroup
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
// @Param group body models.IPGroup true "IP Group"
// @Success 200 {object} models.IPGroup
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Group Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/pools/{id} [put]
func UpdateGroupHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var group models.IPGroup
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
// @Param cursor query int false "Byte offset cursor"
// @Param limit query int false "Number of entries to return"
// @Param search query string false "Search prefix or tag"
// @Success 200 {object} models.IPPoolPreviewResponse
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Group Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/pools/{id}/preview [get]
func PreviewPoolHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cursorStr := r.URL.Query().Get("cursor")
	cursor, _ := strconv.ParseInt(cursorStr, 10, 64)
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
	if err := ipPoolService.DeleteGroup(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// ListExportsHandler godoc
// @Summary List all IP exports
// @Tags network/ip
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name"
// @Success 200 {object} common.PaginatedResponse{items=[]models.IPExport}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/exports [get]
func ListExportsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	items, total, err := ipPoolService.ListExports(r.Context(), page, pageSize, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.PaginatedSuccess(w, r, items, total, page, pageSize)
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
		common.BadRequestError(w, r, http.StatusNotFound, "task not found")
		return
	}
	common.Success(w, r, task)
}

// DownloadExportHandler godoc
// @Summary Download export result
// @Tags network/ip
// @Produce octet-stream
// @Param taskId path string true "Task ID"
// @Success 200 {file} file
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "File Not Found"
// @Security ApiKeyAuth
// @Router /network/ip/exports/download/{taskId} [get]
func DownloadExportHandler(w http.ResponseWriter, r *http.Request) {
	taskId := chi.URLParam(r, "taskId")
	task := exportManager.GetTask(taskId)
	if task == nil || task.Status != "Success" {
		http.Error(w, "file not ready or not found", http.StatusNotFound)
		return
	}

	tempFileName := fmt.Sprintf("export_%s.%s", task.ID, task.Format)
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

// ListSyncPoliciesHandler godoc
// @Summary List all IP sync policies
// @Tags network/ip
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name"
// @Success 200 {object} common.PaginatedResponse{items=[]models.IPSyncPolicy}
// @Failure 401 {object} common.Response "Unauthorized"
// @Security ApiKeyAuth
// @Router /network/ip/sync [get]
func ListSyncPoliciesHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")

	items, total, err := ipPoolService.ListSyncPolicies(r.Context(), page, pageSize, search)
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.PaginatedSuccess(w, r, items, total, page, pageSize)
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
// @Param id path string true "Policy ID"
// @Success 200 {string} string "success"
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
	common.Success(w, r, "success")
}

// IPRouter registers the IP routes
func IPRouter(r chi.Router) {
	r.Route("/network/ip", func(r chi.Router) {
		r.Route("/pools", func(r chi.Router) {
			r.Get("/", ListGroupsHandler)
			r.With(middlewares.RequirePermission("create", "network/ip")).Post("/", CreateGroupHandler)
			r.With(middlewares.RequirePermission("update", "network/ip")).Put("/{id}", UpdateGroupHandler)
			r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/{id}", DeleteGroupHandler)
			r.Get("/{id}/preview", PreviewPoolHandler)
			r.Post("/{id}/entries", ManagePoolEntryHandler)
			r.Delete("/{id}/entries", DeletePoolEntryHandler)
		})
		r.Route("/analysis", func(r chi.Router) {
			r.Post("/hit-test", HitTestHandler)
			r.Get("/info", IPInfoHandler)
		})
		r.Route("/exports", func(r chi.Router) {
			r.Get("/", ListExportsHandler)
			r.With(middlewares.RequirePermission("create", "network/ip")).Post("/", CreateExportHandler)
			r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/{id}", DeleteExportHandler)
			r.Post("/{id}/trigger", TriggerExportHandler)
			r.Get("/task/{taskId}", ExportTaskStatusHandler)
			r.Get("/download/{taskId}", DownloadExportHandler)
		})
		r.Route("/sync", func(r chi.Router) {
			r.Get("/", ListSyncPoliciesHandler)
			r.With(middlewares.RequirePermission("create", "network/ip")).Post("/", CreateSyncPolicyHandler)
			r.With(middlewares.RequirePermission("update", "network/ip")).Put("/{id}", UpdateSyncPolicyHandler)
			r.With(middlewares.RequirePermission("delete", "network/ip")).Delete("/{id}", DeleteSyncPolicyHandler)
			r.Post("/{id}/trigger", TriggerSyncHandler)
		})
	})
}
