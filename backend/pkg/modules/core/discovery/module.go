package discovery

import (
	"context"
	controllercommon "homelab/pkg/controllers"
	discoverycontroller "homelab/pkg/controllers/core/discovery"
	"homelab/pkg/controllers/routerx"
	runtimepkg "homelab/pkg/runtime"
	discoveryservice "homelab/pkg/services/core/discovery"
	"net/http"
)

type Module struct {
	service *discoveryservice.Service
}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "core.discovery" }

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	m.service = discoveryservice.NewService(deps)
	return nil
}

func (m *Module) Routes() runtimepkg.RouteHandler {
	return routerx.New("/discovery",
		routerx.WithScope(routerx.Scope{
			Resource: "discovery",
			Audit:    "discovery",
			UsesAuth: true,
			Extra: []func(http.Handler) http.Handler{
				controllercommon.WithDiscoveryService(m.service),
			},
		}),
		routerx.Routes(
			routerx.Get("/lookup", discoverycontroller.LookupHandler, "list"),
			routerx.Get("/codes", discoverycontroller.ScanCodesHandler, "list"),
		),
	)
}

func (m *Module) Start(context.Context) error { return nil }

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
