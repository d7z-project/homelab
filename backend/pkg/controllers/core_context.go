package controllers

import (
	"net/http"

	discoveryservice "homelab/pkg/services/core/discovery"
)

const discoveryServiceContextKey ContextKey = "controllers.discovery_service"

func WithDiscoveryService(service *discoveryservice.Service) func(http.Handler) http.Handler {
	return WithValue(discoveryServiceContextKey, service)
}

func DiscoveryServiceFromRequest(w http.ResponseWriter, r *http.Request) (*discoveryservice.Service, bool) {
	return ValueFromRequest(w, r, discoveryServiceContextKey, func(service *discoveryservice.Service) bool {
		return service != nil
	})
}
