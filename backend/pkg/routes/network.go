package routes

import (
	networkroutes "homelab/pkg/routes/network"
	"homelab/pkg/services/network/intelligence"
	"homelab/pkg/services/network/ip"
	"homelab/pkg/services/network/site"

	"github.com/go-chi/chi/v5"
)

func RegisterNetworkDNS(r chi.Router) {
	networkroutes.RegisterDNS(r)
}

func RegisterNetworkIP(r chi.Router, poolService *ip.IPPoolService, analysis *ip.AnalysisEngine, exports *ip.ExportManager) {
	networkroutes.RegisterIP(r, poolService, analysis, exports)
}

func RegisterNetworkSite(r chi.Router, poolService *site.SitePoolService, analysis *site.AnalysisEngine, exports *site.ExportManager) {
	networkroutes.RegisterSite(r, poolService, analysis, exports)
}

func RegisterNetworkIntelligence(r chi.Router, service *intelligence.IntelligenceService) {
	networkroutes.RegisterIntelligence(r, service)
}
