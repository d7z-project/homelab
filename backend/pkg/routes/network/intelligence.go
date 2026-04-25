package network

import (
	controllerdeps "homelab/pkg/controllers"
	"homelab/pkg/controllers/middlewares"
	intcontroller "homelab/pkg/controllers/network/intelligence"
	"homelab/pkg/services/network/intelligence"

	"github.com/go-chi/chi/v5"
)

func RegisterIntelligence(r chi.Router, service *intelligence.IntelligenceService) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.AuditMiddleware("network/intelligence"))
		r.Use(controllerdeps.WithIntelligenceControllerDeps(service))

		r.With(middlewares.RequirePermission("list", "network/intelligence")).Get("/api/v1/network/intelligence/sources", intcontroller.ScanIntelligenceSourcesHandler)
		r.With(middlewares.RequirePermission("create", "network/intelligence")).Post("/api/v1/network/intelligence/sources", intcontroller.CreateIntelligenceSourceHandler)
		r.With(middlewares.RequirePermission("update", "network/intelligence")).Put("/api/v1/network/intelligence/sources/{id}", intcontroller.UpdateIntelligenceSourceHandler)
		r.With(middlewares.RequirePermission("delete", "network/intelligence")).Delete("/api/v1/network/intelligence/sources/{id}", intcontroller.DeleteIntelligenceSourceHandler)
		r.With(middlewares.RequirePermission("execute", "network/intelligence")).Post("/api/v1/network/intelligence/sources/{id}/sync", intcontroller.SyncIntelligenceSourceHandler)
		r.With(middlewares.RequirePermission("execute", "network/intelligence")).Post("/api/v1/network/intelligence/sync/{id}/cancel", intcontroller.CancelIntelligenceSyncHandler)
	})
}
