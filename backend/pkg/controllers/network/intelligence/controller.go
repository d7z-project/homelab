package intelligence

import (
	apiv1 "homelab/pkg/apis/network/intelligence/v1"
	"homelab/pkg/common"
	controllercommon "homelab/pkg/controllers"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ScanIntelligenceSourcesHandler godoc
// @Summary Scan intelligence sources
// @Tags network/intelligence
// @Produce json
// @Param cursor query string false "Cursor"
// @Param limit query int false "Limit"
// @Param search query string false "Search"
// @Success 200 {object} common.CursorResponse{items=[]models.IntelligenceSource}
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/intelligence/sources [get]
func ScanIntelligenceSourcesHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.IntelligenceDepsFromRequest(w, r)
	if !ok {
		return
	}
	cursor, limit := controllercommon.GetCursorParams(r)
	search := r.URL.Query().Get("search")
	res, err := deps.Service.ScanSources(r.Context(), cursor, limit, search)
	if err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.CursorSuccess(w, r, mapSources(res))
}

// CreateIntelligenceSourceHandler godoc
// @Summary Create intelligence source
// @Tags network/intelligence
// @Accept json
// @Produce json
// @Param source body models.IntelligenceSource true "Source"
// @Success 200 {object} models.IntelligenceSource
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/intelligence/sources [post]
func CreateIntelligenceSourceHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.IntelligenceDepsFromRequest(w, r)
	if !ok {
		return
	}
	var source apiv1.Source
	if err := render.Bind(r, &source); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	model := toModelSource(source)
	if err := deps.Service.CreateSource(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPISource(model))
}

// UpdateIntelligenceSourceHandler godoc
// @Summary Update intelligence source
// @Tags network/intelligence
// @Accept json
// @Produce json
// @Param id path string true "Source ID"
// @Param source body models.IntelligenceSource true "Source"
// @Success 200 {object} models.IntelligenceSource
// @Failure 400 {object} common.Response "Bad Request"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Not Found"
// @Security ApiKeyAuth
// @Router /network/intelligence/sources/{id} [put]
func UpdateIntelligenceSourceHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.IntelligenceDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var source apiv1.Source
	if err := render.Bind(r, &source); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	source.ID = id
	model := toModelSource(source)
	if err := deps.Service.UpdateSource(r.Context(), &model); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, toAPISource(model))
}

// DeleteIntelligenceSourceHandler godoc
// @Summary Delete intelligence source
// @Tags network/intelligence
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Source Not Found"
// @Security ApiKeyAuth
// @Router /network/intelligence/sources/{id} [delete]
func DeleteIntelligenceSourceHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.IntelligenceDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := deps.Service.DeleteSource(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, "success")
}

// SyncIntelligenceSourceHandler godoc
// @Summary Trigger manual sync
// @Tags network/intelligence
// @Produce json
// @Param id path string true "Source ID"
// @Success 200 {string} string "sync started"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Failure 404 {object} common.Response "Source Not Found"
// @Security ApiKeyAuth
// @Router /network/intelligence/sources/{id}/sync [post]
func SyncIntelligenceSourceHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.IntelligenceDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if err := deps.Service.SyncSource(r.Context(), id); err != nil {
		controllercommon.HandleError(w, r, err)
		return
	}
	common.Success(w, r, "sync started")
}

// CancelIntelligenceSyncHandler godoc
// @Summary Cancel intelligence sync task
// @Tags network/intelligence
// @Produce json
// @Param id path string true "Task ID"
// @Success 200 {string} string "success"
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 404 {object} common.Response "Task Not Found"
// @Security ApiKeyAuth
// @Router /network/intelligence/sync/{id}/cancel [post]
func CancelIntelligenceSyncHandler(w http.ResponseWriter, r *http.Request) {
	deps, ok := controllercommon.IntelligenceDepsFromRequest(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if !deps.Service.CancelTask(id) {
		common.Error(w, r, http.StatusNotFound, http.StatusNotFound, "task not found or not cancelable")
		return
	}
	common.Success(w, r, "success")
}
