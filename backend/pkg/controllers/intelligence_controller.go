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

// @Summary List intelligence sources
// @Tags network/intelligence
// @Success 200 {array} models.IntelligenceSource
// @Router /network/intelligence/sources [get]
func ListIntelligenceSourcesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := intelligenceService.ListSources(r.Context())
	if err != nil { HandleError(w, r, err); return }
	common.Success(w, r, items)
}

// @Summary Create intelligence source
// @Tags network/intelligence
// @Param source body models.IntelligenceSource true "Source"
// @Success 200 {object} models.IntelligenceSource
// @Router /network/intelligence/sources [post]
func CreateIntelligenceSourceHandler(w http.ResponseWriter, r *http.Request) {
	var source models.IntelligenceSource
	if err := render.Bind(r, &source); err != nil { common.BadRequestError(w, r, http.StatusBadRequest, err.Error()); return }
	if err := intelligenceService.CreateSource(r.Context(), &source); err != nil { HandleError(w, r, err); return }
	common.Success(w, r, source)
}

// @Summary Delete intelligence source
// @Tags network/intelligence
// @Param id path string true "Source ID"
// @Success 200 {string} string "success"
// @Router /network/intelligence/sources/{id} [delete]
func DeleteIntelligenceSourceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := intelligenceService.DeleteSource(r.Context(), id); err != nil { HandleError(w, r, err); return }
	common.Success(w, r, "success")
}

// @Summary Trigger manual sync
// @Tags network/intelligence
// @Param id path string true "Source ID"
// @Success 200 {string} string "sync started"
// @Router /network/intelligence/sources/{id}/sync [post]
func SyncIntelligenceSourceHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := intelligenceService.SyncSource(r.Context(), id); err != nil { HandleError(w, r, err); return }
	common.Success(w, r, "sync started")
}

func IntelligenceRouter(r chi.Router) {
	r.Route("/network/intelligence", func(r chi.Router) {
		r.Use(middlewares.RequirePermission("admin", "network/intelligence"))
		r.Get("/sources", ListIntelligenceSourcesHandler)
		r.Post("/sources", CreateIntelligenceSourceHandler)
		r.Delete("/sources/{id}", DeleteIntelligenceSourceHandler)
		r.Post("/sources/{id}/sync", SyncIntelligenceSourceHandler)
	})
}
