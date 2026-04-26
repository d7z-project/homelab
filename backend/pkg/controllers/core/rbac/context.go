package rbac

import (
	"net/http"

	controllercommon "homelab/pkg/controllers"
	discoveryservice "homelab/pkg/services/core/discovery"
)

const discoveryServiceContextKey controllercommon.ContextKey = "controllers.core.rbac.discovery_service"

func WithDiscoveryService(service *discoveryservice.Service) func(http.Handler) http.Handler {
	return controllercommon.WithValue(discoveryServiceContextKey, service)
}

func discoveryServiceFromRequest(w http.ResponseWriter, r *http.Request) (*discoveryservice.Service, bool) {
	return controllercommon.ValueFromRequest(w, r, discoveryServiceContextKey, func(service *discoveryservice.Service) bool {
		return service != nil
	})
}
