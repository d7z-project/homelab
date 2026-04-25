package dns

import (
	"context"
	"homelab/pkg/controllers/middlewares"
	dnscontroller "homelab/pkg/controllers/network/dns"
	runtimepkg "homelab/pkg/runtime"
	dnsservice "homelab/pkg/services/network/dns"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "network.dns" }

func (m *Module) RegisterRoutes(r chi.Router) {
	r.Route("/network/dns", func(r chi.Router) {
		r.Group(func(r chi.Router) {
			r.Use(middlewares.AuthMiddleware)
			r.Use(middlewares.AuditMiddleware("network/dns"))

			r.With(middlewares.RequirePermission("get", "network/dns")).Get("/export", dnscontroller.ExportHandler)

			r.With(middlewares.RequirePermission("list", "network/dns")).Get("/domains", dnscontroller.ScanDomainsHandler)
			r.With(middlewares.RequirePermission("create", "network/dns")).Post("/domains", dnscontroller.CreateDomainHandler)
			r.With(middlewares.RequirePermission("update", "network/dns")).Put("/domains/{id}", dnscontroller.UpdateDomainHandler)
			r.With(middlewares.RequirePermission("delete", "network/dns")).Delete("/domains/{id}", dnscontroller.DeleteDomainHandler)

			r.With(middlewares.RequirePermission("list", "network/dns")).Get("/records", dnscontroller.ScanRecordsHandler)
			r.With(middlewares.RequirePermission("create", "network/dns")).Post("/records", dnscontroller.CreateRecordHandler)
			r.With(middlewares.RequirePermission("update", "network/dns")).Put("/records/{id}", dnscontroller.UpdateRecordHandler)
			r.With(middlewares.RequirePermission("delete", "network/dns")).Delete("/records/{id}", dnscontroller.DeleteRecordHandler)
		})
	})
}

func (m *Module) Start(context.Context) error {
	dnsservice.RegisterDiscovery()
	dnsservice.RegisterActionProcessors()
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
