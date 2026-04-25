package core

import (
	authcontroller "homelab/pkg/controllers/core/auth"
	"homelab/pkg/controllers/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterAuth(r chi.Router) {
	r.Get("/api/v1/ping", middlewares.PingHandler)
	r.Post("/api/v1/login", authcontroller.LoginHandler)

	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Get("/api/v1/info", authcontroller.InfoHandler)
		r.Post("/api/v1/logout", authcontroller.LogoutHandler)
	})
}
