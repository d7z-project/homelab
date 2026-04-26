package auth

import (
	"context"
	authcontroller "homelab/pkg/controllers/core/auth"
	"homelab/pkg/controllers/middlewares"
	runtimepkg "homelab/pkg/runtime"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.auth" }

func (m *Module) Init(runtimepkg.ModuleDeps) error { return nil }

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/auth", func(r chi.Router) {
		r.Get("/ping", middlewares.PingHandler)
		r.Post("/login", authcontroller.LoginHandler)

		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Get("/info", authcontroller.InfoHandler)
			r.Post("/logout", authcontroller.LogoutHandler)
		})
	})
}

func (m *Module) Start(context.Context) error { return nil }

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
