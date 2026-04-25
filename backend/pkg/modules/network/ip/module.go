package ip

import (
	"context"
	"homelab/pkg/routes"
	runtimepkg "homelab/pkg/runtime"
	ipservice "homelab/pkg/services/network/ip"
	ruleservice "homelab/pkg/services/rules"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	service  *ipservice.IPPoolService
	analysis *ipservice.AnalysisEngine
	exports  *ipservice.ExportManager
}

func New(enricher *ipservice.MMDBManager) *Module {
	analysis := ipservice.NewAnalysisEngine(enricher)
	exports := ipservice.NewExportManager(analysis)
	return &Module{
		service:  ipservice.NewIPPoolService(analysis, exports),
		analysis: analysis,
		exports:  exports,
	}
}

func (m *Module) Name() string { return "network.ip" }

func (m *Module) RegisterRoutes(r chi.Router) {
	routes.RegisterNetworkIP(r, m.service, m.analysis, m.exports)
}

func (m *Module) Start(ctx context.Context) error {
	ruleservice.RegisterIPDiscovery()
	m.service.StartSyncRunner(ctx)
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
