package ip

import (
	"net/http"

	controllercommon "homelab/pkg/controllers"
	ipservice "homelab/pkg/services/network/ip"
)

type controllerDeps struct {
	PoolService *ipservice.IPPoolService
	Analysis    *ipservice.AnalysisEngine
	Exports     *ipservice.ExportManager
	TempFS      http.FileSystem
}

const controllerDepsContextKey controllercommon.ContextKey = "controllers.network.ip"

func WithControllerDeps(service *ipservice.IPPoolService, engine *ipservice.AnalysisEngine, exports *ipservice.ExportManager, tempFS http.FileSystem) func(http.Handler) http.Handler {
	deps := controllerDeps{PoolService: service, Analysis: engine, Exports: exports, TempFS: tempFS}
	return controllercommon.WithValue(controllerDepsContextKey, deps)
}

func depsFromRequest(w http.ResponseWriter, r *http.Request) (controllerDeps, bool) {
	return controllercommon.ValueFromRequest(w, r, controllerDepsContextKey, func(deps controllerDeps) bool {
		return deps.PoolService != nil && deps.Analysis != nil && deps.Exports != nil && deps.TempFS != nil
	})
}
