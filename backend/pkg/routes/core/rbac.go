package core

import (
	rbaccontroller "homelab/pkg/controllers/core/rbac"
	"homelab/pkg/controllers/middlewares"

	"github.com/go-chi/chi/v5"
)

func RegisterRBAC(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.AuditMiddleware("rbac"))

		r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/rbac/resources/suggest", rbaccontroller.SuggestResourcesHandler)
		r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/rbac/verbs/suggest", rbaccontroller.SuggestVerbsHandler)
		r.With(middlewares.RequirePermission("simulate", "rbac")).Post("/api/v1/rbac/simulate", rbaccontroller.SimulatePermissionsHandler)

		r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/rbac/serviceaccounts", rbaccontroller.ScanServiceAccountsHandler)
		r.With(middlewares.RequirePermission("create", "rbac")).Post("/api/v1/rbac/serviceaccounts", rbaccontroller.CreateServiceAccountHandler)
		r.With(middlewares.RequirePermission("update", "rbac")).Put("/api/v1/rbac/serviceaccounts/{id}", rbaccontroller.UpdateServiceAccountHandler)
		r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/api/v1/rbac/serviceaccounts/{id}", rbaccontroller.DeleteServiceAccountHandler)
		r.With(middlewares.RequirePermission("update", "rbac")).Post("/api/v1/rbac/serviceaccounts/{id}/reset", rbaccontroller.ResetServiceAccountTokenHandler)

		r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/rbac/roles", rbaccontroller.ScanRolesHandler)
		r.With(middlewares.RequirePermission("create", "rbac")).Post("/api/v1/rbac/roles", rbaccontroller.CreateRoleHandler)
		r.With(middlewares.RequirePermission("update", "rbac")).Put("/api/v1/rbac/roles/{id}", rbaccontroller.UpdateRoleHandler)
		r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/api/v1/rbac/roles/{id}", rbaccontroller.DeleteRoleHandler)

		r.With(middlewares.RequirePermission("list", "rbac")).Get("/api/v1/rbac/rolebindings", rbaccontroller.ScanRoleBindingsHandler)
		r.With(middlewares.RequirePermission("create", "rbac")).Post("/api/v1/rbac/rolebindings", rbaccontroller.CreateRoleBindingHandler)
		r.With(middlewares.RequirePermission("update", "rbac")).Put("/api/v1/rbac/rolebindings/{id}", rbaccontroller.UpdateRoleBindingHandler)
		r.With(middlewares.RequirePermission("delete", "rbac")).Delete("/api/v1/rbac/rolebindings/{id}", rbaccontroller.DeleteRoleBindingHandler)
	})
}
