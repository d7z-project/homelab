package rbac

import (
	"context"
	controllercommon "homelab/pkg/controllers"
	rbaccontroller "homelab/pkg/controllers/core/rbac"
	"homelab/pkg/controllers/routerx"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"
	discoveryservice "homelab/pkg/services/core/discovery"
	rbacservice "homelab/pkg/services/core/rbac"
	"net/http"
)

type Module struct {
	discovery *discoveryservice.Service
	registry  *registryruntime.Registry
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.rbac" }

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	rbacrepo.Configure(deps.DB)
	m.discovery = discoveryservice.NewService(deps)
	m.registry = deps.Registry
	return nil
}

func (m *Module) Routes() runtimepkg.RouteHandler {
	return routerx.New("/rbac",
		routerx.WithScope(routerx.Scope{
			Resource: "rbac",
			Audit:    "rbac",
			UsesAuth: true,
			Extra: []func(http.Handler) http.Handler{
				controllercommon.WithDiscoveryService(m.discovery),
			},
		}),
		routerx.Routes(
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
		),
	)
}

func (m *Module) Start(ctx context.Context) error {
	_ = ctx
	rbacservice.RegisterDiscovery(m.registry)
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
