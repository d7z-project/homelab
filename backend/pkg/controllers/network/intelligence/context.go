package intelligence

import (
	"net/http"

	controllercommon "homelab/pkg/controllers"
	intservice "homelab/pkg/services/network/intelligence"
)

type controllerDeps struct {
	Service *intservice.IntelligenceService
}

const controllerDepsContextKey controllercommon.ContextKey = "controllers.network.intelligence"

func WithControllerDeps(service *intservice.IntelligenceService) func(http.Handler) http.Handler {
	deps := controllerDeps{Service: service}
	return controllercommon.WithValue(controllerDepsContextKey, deps)
}

func depsFromRequest(w http.ResponseWriter, r *http.Request) (controllerDeps, bool) {
	return controllercommon.ValueFromRequest(w, r, controllerDepsContextKey, func(deps controllerDeps) bool {
		return deps.Service != nil
	})
}
