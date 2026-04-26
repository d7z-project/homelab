package routerx

import (
	"net/http"
	"sync"

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

type NodeOption func(*node)

type Handler struct {
	mountPath string
	root      *node
	once      sync.Once
	built     http.Handler
}

type node struct {
	pattern     string
	scope       Scope
	middlewares []func(http.Handler) http.Handler
	routes      []Route
	children    []*node
	mounts      []*mountedHandler
}

type mountedHandler struct {
	pattern string
	handler http.Handler
	scope   Scope
	extra   []func(http.Handler) http.Handler
}

func New(mountPath string, opts ...NodeOption) *Handler {
	root := &node{}
	for _, opt := range opts {
		if opt != nil {
			opt(root)
		}
	}
	return &Handler{mountPath: mountPath, root: root}
}

func Group(pattern string, opts ...NodeOption) NodeOption {
	return func(parent *node) {
		child := &node{pattern: pattern}
		for _, opt := range opts {
			if opt != nil {
				opt(child)
			}
		}
		parent.children = append(parent.children, child)
	}
}

func Mount(pattern string, handler http.Handler, opts ...NodeOption) NodeOption {
	return func(parent *node) {
		mounted := &mountedHandler{
			pattern: pattern,
			handler: handler,
		}
		tmp := &node{}
		for _, opt := range opts {
			if opt != nil {
				opt(tmp)
			}
		}
		mounted.scope = tmp.scope
		mounted.extra = append(mounted.extra, tmp.middlewares...)
		mounted.extra = append(mounted.extra, tmp.scope.Extra...)
		parent.mounts = append(parent.mounts, mounted)
	}
}

func Use(mw ...func(http.Handler) http.Handler) NodeOption {
	return func(n *node) {
		n.middlewares = append(n.middlewares, mw...)
	}
}

func WithScope(scope Scope) NodeOption {
	return func(n *node) {
		n.scope = scope
	}
}

func Routes(routes ...Route) NodeOption {
	return func(n *node) {
		n.routes = append(n.routes, routes...)
	}
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

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.build().ServeHTTP(w, r)
}

func (h *Handler) MountPath() string {
	if h == nil {
		return ""
	}
	return h.mountPath
}

func (h *Handler) build() http.Handler {
	h.once.Do(func() {
		r := chi.NewRouter()
		if h.root != nil {
			bindNode(r, h.root)
		}
		h.built = r
	})
	return h.built
}

func bindNode(r chi.Router, n *node) {
	if n == nil {
		return
	}
	if n.pattern != "" {
		r.Route(n.pattern, func(r chi.Router) {
			applyNode(r, n)
		})
		return
	}
	applyNode(r, n)
}

func applyNode(r chi.Router, n *node) {
	r.Group(func(r chi.Router) {
		applyScope(r, n.scope)
		for _, mw := range n.middlewares {
			r.Use(mw)
		}
		registerRoutes(r, n.scope.Resource, n.routes)
		for _, child := range n.children {
			bindNode(r, child)
		}
		for _, mounted := range n.mounts {
			r.Mount(mounted.pattern, wrapMountedHandler(mounted))
		}
	})
}

func applyScope(r chi.Router, scope Scope) {
	if scope.UsesAuth {
		r.Use(middlewares.AuthMiddleware)
	}
	if scope.Audit != "" {
		r.Use(middlewares.AuditMiddleware(scope.Audit))
	}
	for _, mw := range scope.Extra {
		r.Use(mw)
	}
}

func wrapMountedHandler(mounted *mountedHandler) http.Handler {
	handler := mounted.handler
	if len(mounted.extra) == 0 && !mounted.scope.UsesAuth && mounted.scope.Audit == "" {
		return handler
	}
	r := chi.NewRouter()
	applyScope(r, mounted.scope)
	for _, mw := range mounted.extra {
		r.Use(mw)
	}
	r.Mount("/", handler)
	return r
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
