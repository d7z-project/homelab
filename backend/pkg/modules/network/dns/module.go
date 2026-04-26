package dns

import (
	"context"
	dnscontroller "homelab/pkg/controllers/network/dns"
	"homelab/pkg/controllers/routerx"
	dnsrepo "homelab/pkg/repositories/network/dns"
	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"
	dnsservice "homelab/pkg/services/network/dns"
)

type Module struct{ registry *registryruntime.Registry }

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "network.dns" }

func (m *Module) Init(deps runtimepkg.ModuleDeps) error {
	dnsrepo.Configure(deps.DB)
	m.registry = deps.Registry
	return nil
}

func (m *Module) Routes() runtimepkg.RouteHandler {
	return routerx.New("/network/dns",
		routerx.WithScope(routerx.Scope{
			Resource: "network/dns",
			Audit:    "network/dns",
			UsesAuth: true,
		}),
		routerx.Routes(
			routerx.Get("/export", dnscontroller.ExportHandler, "get"),
			routerx.Get("/domains", dnscontroller.ScanDomainsHandler, "list"),
			routerx.Post("/domains", dnscontroller.CreateDomainHandler, "create"),
			routerx.Put("/domains/{id}", dnscontroller.UpdateDomainHandler, "update"),
			routerx.Delete("/domains/{id}", dnscontroller.DeleteDomainHandler, "delete"),
			routerx.Get("/records", dnscontroller.ScanRecordsHandler, "list"),
			routerx.Post("/records", dnscontroller.CreateRecordHandler, "create"),
			routerx.Put("/records/{id}", dnscontroller.UpdateRecordHandler, "update"),
			routerx.Delete("/records/{id}", dnscontroller.DeleteRecordHandler, "delete"),
		),
	)
}

func (m *Module) Start(ctx context.Context) error {
	_ = ctx
	dnsservice.RegisterDiscovery(m.registry)
	dnsservice.RegisterActionProcessors()
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
