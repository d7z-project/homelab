package runtime

import (
	"context"

	"github.com/go-chi/chi/v5"
)

type Module interface {
	Name() string
	Init(deps ModuleDeps) error
	RegisterRoutes(r chi.Router)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}
