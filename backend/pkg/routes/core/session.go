package core

import (
	authcontroller "homelab/pkg/controllers/core/auth"
	"homelab/pkg/controllers/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterSession(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.AuditMiddleware("rbac"))
		r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/auth/sessions", authcontroller.ScanSessionsHandler)
		r.With(middlewares.RequirePermission("admin", "rbac")).Delete("/api/v1/auth/sessions/{id}", authcontroller.RevokeSessionHandler)
	})
}
