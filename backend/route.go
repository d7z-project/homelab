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
		r.Route("/actions/webhooks/{token}", func(r chi.Router) {
			r.Get("/", controllers.WebhookHandler)
			r.Post("/", controllers.WebhookHandler)
		})

		r.Group(func(r chi.Router) {
			r.Use(render.SetContentType(render.ContentTypeJSON))
			r.Use(middlewares.AuthMiddleware)
			r.Get("/info", controllers.InfoHandler)
			r.Post("/logout", controllers.LogoutHandler)
			r.Get("/network/dns/export", controllers.ExportHandler)
			r.Route("/discovery", controllers.DiscoveryController)

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
				r.Use(middlewares.RequirePermission("admin", "network/dns"))
				r.Use(middlewares.AuditMiddleware("network/dns"))
				controllers.DNSRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequirePermission("admin", "actions"))
				r.Use(middlewares.AuditMiddleware("actions"))
				controllers.ActionsRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequirePermission("admin", "network/ip"))
				r.Use(middlewares.AuditMiddleware("network/ip"))
				controllers.IPRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.RequirePermission("admin", "network/site"))
				r.Use(middlewares.AuditMiddleware("network/site"))
				controllers.SiteRouter(r)
			})

			r.Group(func(r chi.Router) {
				controllers.IntelligenceRouter(r)
			})
		})
	})
}
