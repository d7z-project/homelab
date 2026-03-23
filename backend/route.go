package main

import (
	"homelab/pkg/controllers"
	"homelab/pkg/controllers/middlewares"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func Router(r chi.Router) {
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Get("/api/v1/ping", middlewares.PingHandler)
	r.Post("/api/v1/login", controllers.LoginHandler)
	r.Get("/api/v1/network/ip/exports/download/{taskId}", controllers.DownloadExportHandler)
	r.Get("/api/v1/network/site/exports/download/{taskId}", controllers.DownloadSiteExportHandler)
	r.Get("/api/v1/actions/webhooks/{token}", controllers.WebhookHandler)
	r.Post("/api/v1/actions/webhooks/{token}", controllers.WebhookHandler)

	r.Group(func(r chi.Router) {
		r.Use(render.SetContentType(render.ContentTypeJSON))
		r.Use(middlewares.AuthMiddleware)
		r.Get("/api/v1/info", controllers.InfoHandler)
		r.Post("/api/v1/logout", controllers.LogoutHandler)
		r.Route("/api/v1/discovery", controllers.DiscoveryController)

		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuditMiddleware("rbac"))
			controllers.RBACRouter(r)

			// Root Session Management
			r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/auth/sessions", controllers.ScanSessionsHandler)

			r.With(middlewares.RequirePermission("admin", "rbac")).Delete("/api/v1/auth/sessions/{id}", controllers.RevokeSessionHandler)
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
}
