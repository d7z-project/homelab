package routerx

import (
	"net/http"

	"homelab/pkg/controllers/middlewares"

	"github.com/go-chi/chi/v5"
)

type Scope struct {
	Resource string
	Audit    string
	UsesAuth bool
	Extra    []func(http.Handler) http.Handler
}

type Route struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
	Actions []string
}

func Mount(r chi.Router, prefix string, scope Scope, routes ...Route) {
	r.Route(prefix, func(r chi.Router) {
		r.Group(func(r chi.Router) {
			if scope.UsesAuth {
				r.Use(middlewares.AuthMiddleware)
			}
			if scope.Audit != "" {
				r.Use(middlewares.AuditMiddleware(scope.Audit))
			}
			for _, mw := range scope.Extra {
				r.Use(mw)
			}
			registerRoutes(r, scope.Resource, routes)
		})
	})
}

func Get(path string, handler http.HandlerFunc, actions ...string) Route {
	return Route{Method: http.MethodGet, Path: path, Handler: handler, Actions: actions}
}

func Post(path string, handler http.HandlerFunc, actions ...string) Route {
	return Route{Method: http.MethodPost, Path: path, Handler: handler, Actions: actions}
}

func Put(path string, handler http.HandlerFunc, actions ...string) Route {
	return Route{Method: http.MethodPut, Path: path, Handler: handler, Actions: actions}
}

func Patch(path string, handler http.HandlerFunc, actions ...string) Route {
	return Route{Method: http.MethodPatch, Path: path, Handler: handler, Actions: actions}
}

func Delete(path string, handler http.HandlerFunc, actions ...string) Route {
	return Route{Method: http.MethodDelete, Path: path, Handler: handler, Actions: actions}
}

func registerRoutes(r chi.Router, resource string, routes []Route) {
	for _, route := range routes {
		handler := http.Handler(route.Handler)
		if len(route.Actions) > 0 {
			handler = middlewares.RequireAnyPermission(resource, route.Actions...)(handler)
		}
		r.Method(route.Method, route.Path, handler)
	}
}
