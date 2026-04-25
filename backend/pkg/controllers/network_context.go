package controllers

import (
	"context"
	"errors"
	"net/http"

	intservice "homelab/pkg/services/network/intelligence"
	ipservice "homelab/pkg/services/network/ip"
	siteservice "homelab/pkg/services/network/site"
)

type controllerContextKey string

const (
	ipControllerDepsKey           controllerContextKey = "controllers.ip"
	siteControllerDepsKey         controllerContextKey = "controllers.site"
	intelligenceControllerDepsKey controllerContextKey = "controllers.intelligence"
)

var errControllerDependenciesNotConfigured = errors.New("controller dependencies not configured")

type IPControllerDeps struct {
	PoolService *ipservice.IPPoolService
	Analysis    *ipservice.AnalysisEngine
	Exports     *ipservice.ExportManager
}

type SiteControllerDeps struct {
	PoolService *siteservice.SitePoolService
	Analysis    *siteservice.AnalysisEngine
	Exports     *siteservice.ExportManager
}

type IntelligenceControllerDeps struct {
	Service *intservice.IntelligenceService
}

func WithIPControllerDeps(service *ipservice.IPPoolService, engine *ipservice.AnalysisEngine, exports *ipservice.ExportManager) func(http.Handler) http.Handler {
	deps := IPControllerDeps{PoolService: service, Analysis: engine, Exports: exports}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), ipControllerDepsKey, deps)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func WithSiteControllerDeps(service *siteservice.SitePoolService, engine *siteservice.AnalysisEngine, exports *siteservice.ExportManager) func(http.Handler) http.Handler {
	deps := SiteControllerDeps{PoolService: service, Analysis: engine, Exports: exports}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), siteControllerDepsKey, deps)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func WithIntelligenceControllerDeps(service *intservice.IntelligenceService) func(http.Handler) http.Handler {
	deps := IntelligenceControllerDeps{Service: service}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), intelligenceControllerDepsKey, deps)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func IPDepsFromRequest(w http.ResponseWriter, r *http.Request) (IPControllerDeps, bool) {
	deps, ok := r.Context().Value(ipControllerDepsKey).(IPControllerDeps)
	if !ok || deps.PoolService == nil || deps.Analysis == nil || deps.Exports == nil {
		HandleError(w, r, errControllerDependenciesNotConfigured)
		return IPControllerDeps{}, false
	}
	return deps, true
}

func SiteDepsFromRequest(w http.ResponseWriter, r *http.Request) (SiteControllerDeps, bool) {
	deps, ok := r.Context().Value(siteControllerDepsKey).(SiteControllerDeps)
	if !ok || deps.PoolService == nil || deps.Analysis == nil || deps.Exports == nil {
		HandleError(w, r, errControllerDependenciesNotConfigured)
		return SiteControllerDeps{}, false
	}
	return deps, true
}

func IntelligenceDepsFromRequest(w http.ResponseWriter, r *http.Request) (IntelligenceControllerDeps, bool) {
	deps, ok := r.Context().Value(intelligenceControllerDepsKey).(IntelligenceControllerDeps)
	if !ok || deps.Service == nil {
		HandleError(w, r, errControllerDependenciesNotConfigured)
		return IntelligenceControllerDeps{}, false
	}
	return deps, true
}
