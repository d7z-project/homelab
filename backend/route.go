package main

import (
	"homelab/pkg/routers"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func Router(r chi.Router) {
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/ping", routers.PingHandler)
		r.Post("/login", routers.LoginHandler)
	})
}
