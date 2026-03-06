package controllers

import (
	"fmt"
	"io"
	"homelab/pkg/common"
	"homelab/pkg/controllers/middlewares"
	"homelab/pkg/models"
	siteservice "homelab/pkg/services/site"
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
// @Summary List all site groups
// @Tags network/site
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name"
// @Success 200 {object} common.PaginatedResponse{items=[]models.SiteGroup}
// @Router /network/site/pools [get]
func ListSiteGroupsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")
	items, total, err := sitePoolService.ListGroups(r.Context(), page, pageSize, search)
	if err != nil { HandleError(w, r, err); return }
	common.PaginatedSuccess(w, r, items, total, page, pageSize)
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
	if err := render.Bind(r, &group); err != nil { common.BadRequestError(w, r, http.StatusBadRequest, err.Error()); return }
	if err := sitePoolService.CreateGroup(r.Context(), &group); err != nil { HandleError(w, r, err); return }
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
	if err := sitePoolService.DeleteGroup(r.Context(), id); err != nil { HandleError(w, r, err); return }
	common.Success(w, r, "success")
}

// PreviewSitePoolHandler godoc
// @Summary Preview site pool data (cursor-based)
// @Tags network/site
// @Produce json
// @Param id path string true "Group ID"
// @Param cursor query int false "Byte offset cursor"
// @Param limit query int false "Number of entries to return"
// @Param search query string false "Search prefix or tag"
// @Success 200 {object} models.SitePoolPreviewResponse
// @Router /network/site/pools/{id}/preview [get]
func PreviewSitePoolHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	cursor, _ := strconv.ParseInt(r.URL.Query().Get("cursor"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 { limit = 100 }
	search := r.URL.Query().Get("search")
	res, err := sitePoolService.PreviewPool(r.Context(), id, cursor, limit, search)
	if err != nil { HandleError(w, r, err); return }
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
	var req models.SitePoolEntryRequest
	if err := render.Bind(r, &req); err != nil { common.BadRequestError(w, r, http.StatusBadRequest, err.Error()); return }
	if err := sitePoolService.ManagePoolEntry(r.Context(), id, &req, "add"); err != nil { HandleError(w, r, err); return }
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
	val := r.URL.Query().Get("value")
	t, _ := strconv.Atoi(r.URL.Query().Get("type"))
	req := models.SitePoolEntryRequest{Type: uint8(t), Value: val}
	if err := sitePoolService.ManagePoolEntry(r.Context(), id, &req, "delete"); err != nil { HandleError(w, r, err); return }
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
	if err := render.DecodeJSON(r.Body, &req); err != nil { common.BadRequestError(w, r, http.StatusBadRequest, err.Error()); return }
	res, err := siteAnalysisEngine.HitTest(r.Context(), req.Domain, req.GroupIDs)
	if err != nil { HandleError(w, r, err); return }
	common.Success(w, r, res)
}

// Site Exports

// ListSiteExportsHandler godoc
// @Summary List all site exports
// @Tags network/site
// @Produce json
// @Param page query int false "Page number"
// @Param pageSize query int false "Items per page"
// @Param search query string false "Search by name"
// @Success 200 {object} common.PaginatedResponse{items=[]models.SiteExport}
// @Router /network/site/exports [get]
func ListSiteExportsHandler(w http.ResponseWriter, r *http.Request) {
	page, pageSize := getPaginationParams(r)
	search := r.URL.Query().Get("search")
	items, total, err := sitePoolService.ListExports(r.Context(), page, pageSize, search)
	if err != nil { HandleError(w, r, err); return }
	common.PaginatedSuccess(w, r, items, total, page, pageSize)
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
	if err := render.Bind(r, &export); err != nil { common.BadRequestError(w, r, http.StatusBadRequest, err.Error()); return }
	if err := sitePoolService.CreateExport(r.Context(), &export); err != nil { HandleError(w, r, err); return }
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
	if err := sitePoolService.DeleteExport(r.Context(), id); err != nil { HandleError(w, r, err); return }
	common.Success(w, r, "success")
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
	if format == "" { format = "text" }
	taskID, err := siteExportManager.TriggerExport(r.Context(), id, format)
	if err != nil { HandleError(w, r, err); return }
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
	if task == nil { common.BadRequestError(w, r, http.StatusNotFound, "task not found"); return }
	common.Success(w, r, task)
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
	if task == nil || task.Status != "Success" { http.Error(w, "file not ready", http.StatusNotFound); return }
	tempFileName := fmt.Sprintf("site_export_%s.%s", task.ID, task.Format)
	f, err := common.TempDir.Open(filepath.Join("temp", tempFileName))
	if err != nil { http.Error(w, "file not found", http.StatusNotFound); return }
	defer f.Close()
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", tempFileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	io.Copy(w, f)
}

func SiteRouter(r chi.Router) {
	r.Route("/network/site", func(r chi.Router) {
		r.Route("/pools", func(r chi.Router) {
			r.Get("/", ListSiteGroupsHandler)
			r.With(middlewares.RequirePermission("create", "network/site")).Post("/", CreateSiteGroupHandler)
			r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/{id}", DeleteSiteGroupHandler)
			r.Get("/{id}/preview", PreviewSitePoolHandler)
			r.Post("/{id}/entries", ManageSitePoolEntryHandler)
			r.Delete("/{id}/entries", DeleteSitePoolEntryHandler)
		})
		r.Route("/analysis", func(r chi.Router) {
			r.Post("/hit-test", SiteHitTestHandler)
		})
		r.Route("/exports", func(r chi.Router) {
			r.Get("/", ListSiteExportsHandler)
			r.With(middlewares.RequirePermission("create", "network/site")).Post("/", CreateSiteExportHandler)
			r.With(middlewares.RequirePermission("delete", "network/site")).Delete("/{id}", DeleteSiteExportHandler)
			r.Post("/{id}/trigger", TriggerSiteExportHandler)
			r.Get("/task/{taskId}", SiteExportTaskStatusHandler)
			r.Get("/download/{taskId}", DownloadSiteExportHandler)
		})
	})
}
