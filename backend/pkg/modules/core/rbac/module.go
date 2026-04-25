package rbac

import (
	"context"
	rbaccontroller "homelab/pkg/controllers/core/rbac"
	"homelab/pkg/controllers/middlewares"
	runtimepkg "homelab/pkg/runtime"
	rbacservice "homelab/pkg/services/core/rbac"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.rbac" }

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/rbac", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AuditMiddleware("rbac"))

			r.With(middlewares.RequirePermission("list", "rbac")).Get("/resources/suggest", rbaccontroller.SuggestResourcesHandler)
			r.With(middlewares.RequirePermission("list", "rbac")).Get("/verbs/suggest", rbaccontroller.SuggestVerbsHandler)
			r.With(middlewares.RequirePermission("simulate", "rbac")).Post("/simulate", rbaccontroller.SimulatePermissionsHandler)

			r.With(middlewares.RequirePermission("list", "rbac")).Get("/serviceaccounts", rbaccontroller.ScanServiceAccountsHandler)
			r.With(middlewares.RequirePermission("create", "rbac")).Post("/serviceaccounts", rbaccontroller.CreateServiceAccountHandler)
			r.With(middlewares.RequirePermission("update", "rbac")).Put("/serviceaccounts/{id}", rbaccontroller.UpdateServiceAccountHandler)
			r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/serviceaccounts/{id}", rbaccontroller.DeleteServiceAccountHandler)
			r.With(middlewares.RequirePermission("update", "rbac")).Post("/serviceaccounts/{id}/reset", rbaccontroller.ResetServiceAccountTokenHandler)

			r.With(middlewares.RequirePermission("list", "rbac")).Get("/roles", rbaccontroller.ScanRolesHandler)
			r.With(middlewares.RequirePermission("create", "rbac")).Post("/roles", rbaccontroller.CreateRoleHandler)
			r.With(middlewares.RequirePermission("update", "rbac")).Put("/roles/{id}", rbaccontroller.UpdateRoleHandler)
			r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/roles/{id}", rbaccontroller.DeleteRoleHandler)

			r.With(middlewares.RequirePermission("list", "rbac")).Get("/rolebindings", rbaccontroller.ScanRoleBindingsHandler)
			r.With(middlewares.RequirePermission("create", "rbac")).Post("/rolebindings", rbaccontroller.CreateRoleBindingHandler)
			r.With(middlewares.RequirePermission("update", "rbac")).Put("/rolebindings/{id}", rbaccontroller.UpdateRoleBindingHandler)
			r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/rolebindings/{id}", rbaccontroller.DeleteRoleBindingHandler)
		})
	})
}

func (m *Module) Start(context.Context) error {
	rbacservice.RegisterDiscovery()
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
