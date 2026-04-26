package site

import (
	"net/http"

	controllercommon "homelab/pkg/controllers"
	siteservice "homelab/pkg/services/network/site"
)

type controllerDeps struct {
	PoolService *siteservice.SitePoolService
	Analysis    *siteservice.AnalysisEngine
	Exports     *siteservice.ExportManager
	TempFS      http.FileSystem
}

const controllerDepsContextKey controllercommon.ContextKey = "controllers.network.site"

func WithControllerDeps(service *siteservice.SitePoolService, engine *siteservice.AnalysisEngine, exports *siteservice.ExportManager, tempFS http.FileSystem) func(http.Handler) http.Handler {
	deps := controllerDeps{PoolService: service, Analysis: engine, Exports: exports, TempFS: tempFS}
	return controllercommon.WithValue(controllerDepsContextKey, deps)
}

func depsFromRequest(w http.ResponseWriter, r *http.Request) (controllerDeps, bool) {
	return controllercommon.ValueFromRequest(w, r, controllerDepsContextKey, func(deps controllerDeps) bool {
		return deps.PoolService != nil && deps.Analysis != nil && deps.Exports != nil && deps.TempFS != nil
	})
}
