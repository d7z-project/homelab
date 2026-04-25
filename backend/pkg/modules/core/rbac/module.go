package rbac

import (
	"context"
	rbaccontroller "homelab/pkg/controllers/core/rbac"
	"homelab/pkg/controllers/routerx"
	runtimepkg "homelab/pkg/runtime"
	rbacservice "homelab/pkg/services/core/rbac"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.rbac" }

func (m *Module) RegisterRoutes(r chi.Router) {
	routerx.Mount(r, "/rbac", routerx.Scope{
		Resource: "rbac",
		Audit:    "rbac",
		UsesAuth: true,
	},
		routerx.Get("/resources/suggest", rbaccontroller.SuggestResourcesHandler, "list"),
		routerx.Get("/verbs/suggest", rbaccontroller.SuggestVerbsHandler, "list"),
		routerx.Post("/simulate", rbaccontroller.SimulatePermissionsHandler, "simulate"),
		routerx.Get("/serviceaccounts", rbaccontroller.ScanServiceAccountsHandler, "list"),
		routerx.Post("/serviceaccounts", rbaccontroller.CreateServiceAccountHandler, "create"),
		routerx.Put("/serviceaccounts/{id}", rbaccontroller.UpdateServiceAccountHandler, "update"),
		routerx.Delete("/serviceaccounts/{id}", rbaccontroller.DeleteServiceAccountHandler, "delete"),
		routerx.Post("/serviceaccounts/{id}/reset", rbaccontroller.ResetServiceAccountTokenHandler, "update"),
		routerx.Get("/roles", rbaccontroller.ScanRolesHandler, "list"),
		routerx.Post("/roles", rbaccontroller.CreateRoleHandler, "create"),
		routerx.Put("/roles/{id}", rbaccontroller.UpdateRoleHandler, "update"),
		routerx.Delete("/roles/{id}", rbaccontroller.DeleteRoleHandler, "delete"),
		routerx.Get("/rolebindings", rbaccontroller.ScanRoleBindingsHandler, "list"),
		routerx.Post("/rolebindings", rbaccontroller.CreateRoleBindingHandler, "create"),
		routerx.Put("/rolebindings/{id}", rbaccontroller.UpdateRoleBindingHandler, "update"),
		routerx.Delete("/rolebindings/{id}", rbaccontroller.DeleteRoleBindingHandler, "delete"),
	)
}

func (m *Module) Start(context.Context) error {
	rbacservice.RegisterDiscovery()
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
