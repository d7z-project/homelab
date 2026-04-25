package core

import (
	auditcontroller "homelab/pkg/controllers/core/audit"
	"homelab/pkg/controllers/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterAudit(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.AuditMiddleware("audit"))
		r.With(middlewares.RequirePermission("list", "audit")).Get("/api/v1/audit/logs", auditcontroller.ScanAuditLogsHandler)
		r.With(middlewares.RequirePermission("delete", "audit")).Post("/api/v1/audit/logs/cleanup", auditcontroller.CleanupAuditLogsHandler)
	})
}
