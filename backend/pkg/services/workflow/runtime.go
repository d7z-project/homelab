package workflow

import (
	"context"
	"errors"
	"net/http"

	runtimepkg "homelab/pkg/runtime"

	"github.com/spf13/afero"
)

type runtimeContextKey string

const workflowRuntimeContextKey runtimeContextKey = "workflow.runtime"

type Runtime struct {
	Deps           runtimepkg.ModuleDeps
	Executor       *Executor
	TriggerManager *TriggerManager
	ActionsFS      afero.Fs
	LogFS          afero.Fs
}

func NewRuntime(deps runtimepkg.ModuleDeps) (*Runtime, error) {
	if deps.FS == nil || deps.TempFS == nil {
		return nil, errors.New("workflow filesystems are not configured")
	}
	_ = deps.TempFS.MkdirAll(ActionsSubDir, 0755)
	_ = deps.FS.MkdirAll(LogSubDir, 0755)
	rt := &Runtime{
		Deps:      deps,
		Executor:  &Executor{},
		ActionsFS: afero.NewBasePathFs(deps.TempFS, ActionsSubDir),
		LogFS:     afero.NewBasePathFs(deps.FS, LogSubDir),
	}
	rt.TriggerManager = NewTriggerManager(rt)
	return rt, nil
}

func (rt *Runtime) WithContext(ctx context.Context) context.Context {
	ctx = rt.Deps.WithContext(ctx)
	return context.WithValue(ctx, workflowRuntimeContextKey, rt)
}

func RuntimeFromContext(ctx context.Context) (*Runtime, bool) {
	rt, ok := ctx.Value(workflowRuntimeContextKey).(*Runtime)
	return rt, ok
}

func MustRuntime(ctx context.Context) *Runtime {
	rt, ok := RuntimeFromContext(ctx)
	if !ok || rt == nil {
		panic("workflow runtime not configured")
	}
	return rt
}

func ContextMiddleware(rt *Runtime) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(rt.WithContext(r.Context())))
		})
	}
}
