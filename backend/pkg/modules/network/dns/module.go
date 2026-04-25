package dns

import (
	"context"
	dnscontroller "homelab/pkg/controllers/network/dns"
	"homelab/pkg/controllers/routerx"
	runtimepkg "homelab/pkg/runtime"
	dnsservice "homelab/pkg/services/network/dns"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "network.dns" }

func (m *Module) RegisterRoutes(r chi.Router) {
	routerx.Mount(r, "/network/dns", routerx.Scope{
		Resource: "network/dns",
		Audit:    "network/dns",
		UsesAuth: true,
	},
		routerx.Get("/export", dnscontroller.ExportHandler, "get"),
		routerx.Get("/domains", dnscontroller.ScanDomainsHandler, "list"),
		routerx.Post("/domains", dnscontroller.CreateDomainHandler, "create"),
		routerx.Put("/domains/{id}", dnscontroller.UpdateDomainHandler, "update"),
		routerx.Delete("/domains/{id}", dnscontroller.DeleteDomainHandler, "delete"),
		routerx.Get("/records", dnscontroller.ScanRecordsHandler, "list"),
		routerx.Post("/records", dnscontroller.CreateRecordHandler, "create"),
		routerx.Put("/records/{id}", dnscontroller.UpdateRecordHandler, "update"),
		routerx.Delete("/records/{id}", dnscontroller.DeleteRecordHandler, "delete"),
	)
}

func (m *Module) Start(context.Context) error {
	dnsservice.RegisterDiscovery()
	dnsservice.RegisterActionProcessors()
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
