package dns

import (
	"context"
	"homelab/pkg/routes"
	runtimepkg "homelab/pkg/runtime"
	dnsservice "homelab/pkg/services/network/dns"

	"github.com/go-chi/chi/v5"
)

type Module struct{}

func New() *Module { return &Module{} }

func (m *Module) Name() string { return "network.dns" }

func (m *Module) RegisterRoutes(r chi.Router) { routes.RegisterNetworkDNS(r) }

func (m *Module) Start(context.Context) error {
	dnsservice.RegisterDiscovery()
	dnsservice.RegisterActionProcessors()
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
