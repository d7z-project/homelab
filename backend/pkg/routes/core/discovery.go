package core

import (
	discoverycontroller "homelab/pkg/controllers/core/discovery"
	"homelab/pkg/controllers/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterDiscovery(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Route("/api/v1/discovery", discoverycontroller.DiscoveryController)
	})
}
