package site

import (
	"context"
	"homelab/pkg/routes"
	runtimepkg "homelab/pkg/runtime"
	ipservice "homelab/pkg/services/network/ip"
	ruleservice "homelab/pkg/services/rules"
	siteservice "homelab/pkg/services/network/site"

	"github.com/go-chi/chi/v5"
)

type Module struct {
	service  *siteservice.SitePoolService
	analysis *siteservice.AnalysisEngine
	exports  *siteservice.ExportManager
}

func New(enricher *ipservice.MMDBManager) *Module {
	analysis := siteservice.NewAnalysisEngine(enricher)
	exports := siteservice.NewExportManager(analysis)
	return &Module{
		service:  siteservice.NewSitePoolService(analysis, exports),
		analysis: analysis,
		exports:  exports,
	}
}

func (m *Module) Name() string { return "network.site" }

func (m *Module) RegisterRoutes(r chi.Router) {
	routes.RegisterNetworkSite(r, m.service, m.analysis, m.exports)
}

func (m *Module) Start(ctx context.Context) error {
	ruleservice.RegisterSiteDiscovery()
	siteservice.RegisterSiteProcessors(m.service)
	m.service.Start(ctx)
	return nil
}

func (m *Module) Stop(context.Context) error { return nil }

var _ runtimepkg.Module = (*Module)(nil)
