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
		r.Get("/network/ip/exports/download/{taskId}", controllers.DownloadExportHandler)
		r.Get("/network/site/exports/download/{taskId}", controllers.DownloadSiteExportHandler)
		r.Route("/actions/webhooks/{token}", func(r chi.Router) {
			r.Get("/", controllers.WebhookHandler)
			r.Post("/", controllers.WebhookHandler)
		})

		r.Group(func(r chi.Router) {
			r.Use(render.SetContentType(render.ContentTypeJSON))
			r.Use(middlewares.AuthMiddleware)
			r.Get("/info", controllers.InfoHandler)
			r.Post("/logout", controllers.LogoutHandler)
			r.Route("/discovery", controllers.DiscoveryController)

			r.Group(func(r chi.Router) {
				r.Use(middlewares.AuditMiddleware("rbac"))
				controllers.RBACRouter(r)

				// Root Session Management
				r.With(middlewares.RequirePermission("list", "rbac")).Get("/auth/sessions", controllers.ScanSessionsHandler)

				r.With(middlewares.RequirePermission("admin", "rbac")).Delete("/auth/sessions/{id}", controllers.RevokeSessionHandler)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.AuditMiddleware("audit"))
				controllers.AuditRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.AuditMiddleware("network/dns"))
				controllers.DNSRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.AuditMiddleware("actions"))
				controllers.ActionsRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.AuditMiddleware("network/ip"))
				controllers.IPRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.AuditMiddleware("network/site"))
				controllers.SiteRouter(r)
			})

			r.Group(func(r chi.Router) {
				r.Use(middlewares.AuditMiddleware("network/intelligence"))
				controllers.IntelligenceRouter(r)
			})
		})
	})
}
