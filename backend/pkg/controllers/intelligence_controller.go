package controllers

import (
	"homelab/pkg/common"
	"homelab/pkg/controllers/middlewares"
	"homelab/pkg/models"
	intservice "homelab/pkg/services/intelligence"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

var intelligenceService *intservice.IntelligenceService

func InitIntelligenceControllers(service *intservice.IntelligenceService) {
	intelligenceService = service
}

// ListIntelligenceSourcesHandler godoc
// @Summary List intelligence sources
// @Tags network/intelligence
// @Produce json
// @Success 200 {array} models.IntelligenceSource
// @Failure 401 {object} common.Response "Unauthorized"
// @Failure 403 {object} common.Response "Forbidden"
// @Security ApiKeyAuth
// @Router /network/intelligence/sources [get]
func ListIntelligenceSourcesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := intelligenceService.ListSources(r.Context())
	if err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, items)
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
	var source models.IntelligenceSource
	if err := render.Bind(r, &source); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	if err := intelligenceService.CreateSource(r.Context(), &source); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, source)
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
	id := chi.URLParam(r, "id")
	var source models.IntelligenceSource
	if err := render.Bind(r, &source); err != nil {
		common.BadRequestError(w, r, http.StatusBadRequest, err.Error())
		return
	}
	source.ID = id
	if err := intelligenceService.UpdateSource(r.Context(), &source); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, source)
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
	id := chi.URLParam(r, "id")
	if err := intelligenceService.DeleteSource(r.Context(), id); err != nil {
		HandleError(w, r, err)
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
	id := chi.URLParam(r, "id")
	if err := intelligenceService.SyncSource(r.Context(), id); err != nil {
		HandleError(w, r, err)
		return
	}
	common.Success(w, r, "sync started")
}

func IntelligenceRouter(r chi.Router) {
	r.Route("/network/intelligence", func(r chi.Router) {
		r.With(middlewares.RequirePermission("list", "network/intelligence")).Get("/sources", ListIntelligenceSourcesHandler)
		r.With(middlewares.RequirePermission("create", "network/intelligence")).Post("/sources", CreateIntelligenceSourceHandler)
		r.With(middlewares.RequirePermission("update", "network/intelligence")).Put("/sources/{id}", UpdateIntelligenceSourceHandler)
		r.With(middlewares.RequirePermission("delete", "network/intelligence")).Delete("/sources/{id}", DeleteIntelligenceSourceHandler)
		r.With(middlewares.RequirePermission("execute", "network/intelligence")).Post("/sources/{id}/sync", SyncIntelligenceSourceHandler)
	})
}
