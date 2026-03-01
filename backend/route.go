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

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequirePermission("admin", "rbac"))
				r.Use(middlewares.AuditMiddleware("rbac"))
				r.Get("/rbac/resources/suggest", controllers.SuggestResourcesHandler)
				controllers.RBACRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequirePermission("admin", "audit"))
				r.Get("/audit/logs", controllers.ListAuditLogsHandler)
			})
		})
	})
}
