package network

import (
	"homelab/pkg/controllers/middlewares"
	dnscontroller "homelab/pkg/controllers/network/dns"

	"github.com/go-chi/chi/v5"
)

func RegisterDNS(r chi.Router) {
	r.Group(func(r chi.Router) {
		r.Use(middlewares.AuthMiddleware)
		r.Use(middlewares.AuditMiddleware("network/dns"))

		r.With(middlewares.RequirePermission("get", "network/dns")).Get("/api/v1/network/dns/export", dnscontroller.ExportHandler)

		r.With(middlewares.RequirePermission("list", "network/dns")).Get("/api/v1/network/dns/domains", dnscontroller.ScanDomainsHandler)
		r.With(middlewares.RequirePermission("create", "network/dns")).Post("/api/v1/network/dns/domains", dnscontroller.CreateDomainHandler)
		r.With(middlewares.RequirePermission("update", "network/dns")).Put("/api/v1/network/dns/domains/{id}", dnscontroller.UpdateDomainHandler)
		r.With(middlewares.RequirePermission("delete", "network/dns")).Delete("/api/v1/network/dns/domains/{id}", dnscontroller.DeleteDomainHandler)

		r.With(middlewares.RequirePermission("list", "network/dns")).Get("/api/v1/network/dns/records", dnscontroller.ScanRecordsHandler)
		r.With(middlewares.RequirePermission("create", "network/dns")).Post("/api/v1/network/dns/records", dnscontroller.CreateRecordHandler)
		r.With(middlewares.RequirePermission("update", "network/dns")).Put("/api/v1/network/dns/records/{id}", dnscontroller.UpdateRecordHandler)
		r.With(middlewares.RequirePermission("delete", "network/dns")).Delete("/api/v1/network/dns/records/{id}", dnscontroller.DeleteRecordHandler)
	})
}
