package main

import (
	"homelab/pkg/controllers"
	"homelab/pkg/controllers/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func Router(r chi.Router) {
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", middlewares.PingHandler)
		r.Post("/login", controllers.LoginHandler)

		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Get("/info", controllers.InfoHandler)
			r.Post("/logout", controllers.LogoutHandler)
			r.Get("/dns/export", controllers.ExportHandler)

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequirePermission("admin", "rbac"))
				r.Use(middlewares.AuditMiddleware("rbac"))
				controllers.RBACRouter(r)

				// Root Session Management
				r.Get("/auth/sessions", controllers.ListSessionsHandler)
				r.Delete("/auth/sessions/{id}", controllers.RevokeSessionHandler)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequirePermission("*", "audit"))
				r.Use(middlewares.AuditMiddleware("audit"))
				controllers.AuditRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequirePermission("admin", "dns"))
				r.Use(middlewares.AuditMiddleware("dns"))
				controllers.DNSRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequirePermission("admin", "orchestration"))
				r.Use(middlewares.AuditMiddleware("orchestration"))
				controllers.OrchestrationRouter(r)
			})
		})
	})
}
