package runtime

import (
	"context"
	"net/http"

	registryruntime "homelab/pkg/runtime/registry"

	"github.com/spf13/afero"
	"gopkg.d7z.net/middleware/kv"
	"gopkg.d7z.net/middleware/lock"
	"gopkg.d7z.net/middleware/subscribe"
)

type contextKey string

const moduleDepsContextKey contextKey = "runtime.moduleDeps"

type ModuleDeps struct {
	Dependencies
	Registry *registryruntime.Registry
}

func (d ModuleDeps) WithContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, moduleDepsContextKey, d)
}

func DetachContext(ctx context.Context) context.Context {
	deps, ok := ModuleDepsFromContext(ctx)
	if !ok {
		return context.Background()
	}
	return deps.WithContext(context.Background())
}

func ContextMiddleware(deps ModuleDeps) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(deps.WithContext(r.Context())))
		})
	}
}

func ModuleDepsFromContext(ctx context.Context) (ModuleDeps, bool) {
	deps, ok := ctx.Value(moduleDepsContextKey).(ModuleDeps)
	return deps, ok
}

func MustModuleDeps(ctx context.Context) ModuleDeps {
	deps, ok := ModuleDepsFromContext(ctx)
	if !ok {
		panic("runtime module dependencies not found in context")
	}
	return deps
}

func DBFromContext(ctx context.Context) kv.KV {
	deps, ok := ModuleDepsFromContext(ctx)
	if !ok {
		return nil
	}
	return deps.DB
}

func LockerFromContext(ctx context.Context) lock.Locker {
	deps, ok := ModuleDepsFromContext(ctx)
	if !ok {
		return nil
	}
	return deps.Locker
}

func SubscriberFromContext(ctx context.Context) subscribe.Subscriber {
	deps, ok := ModuleDepsFromContext(ctx)
	if !ok {
		return nil
	}
	return deps.Subscriber
}

func FSFromContext(ctx context.Context) afero.Fs {
	deps, ok := ModuleDepsFromContext(ctx)
	if !ok {
		return nil
	}
	return deps.FS
}

func TempFSFromContext(ctx context.Context) afero.Fs {
	deps, ok := ModuleDepsFromContext(ctx)
	if !ok {
		return nil
	}
	return deps.TempFS
}

func RegistryFromContext(ctx context.Context) *registryruntime.Registry {
	deps, ok := ModuleDepsFromContext(ctx)
	if !ok {
		return nil
	}
	return deps.Registry
}
