package runtime

import (
	"context"
	"net/http"
)

type RouteHandler interface {
	http.Handler
	MountPath() string
}

type Module interface {
	Name() string
	Init(deps ModuleDeps) error
	Routes() RouteHandler
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
