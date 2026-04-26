package discovery

import (
	"net/http"

	controllercommon "homelab/pkg/controllers"
	discoveryservice "homelab/pkg/services/core/discovery"
)

const serviceContextKey controllercommon.ContextKey = "controllers.core.discovery.service"

func WithService(service *discoveryservice.Service) func(http.Handler) http.Handler {
	return controllercommon.WithValue(serviceContextKey, service)
}

func serviceFromRequest(w http.ResponseWriter, r *http.Request) (*discoveryservice.Service, bool) {
	return controllercommon.ValueFromRequest(w, r, serviceContextKey, func(service *discoveryservice.Service) bool {
		return service != nil
	})
}
