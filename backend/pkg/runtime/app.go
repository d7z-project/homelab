package runtime

import (
	"context"
	"errors"
	"fmt"
	"sync"

	registryruntime "homelab/pkg/runtime/registry"

	"github.com/go-chi/chi/v5"
	"github.com/spf13/afero"
	"gopkg.d7z.net/middleware/kv"
	"gopkg.d7z.net/middleware/lock"
	"gopkg.d7z.net/middleware/subscribe"
)

type Dependencies struct {
	DB         kv.KV
	Locker     lock.Locker
	Subscriber subscribe.Subscriber
	FS         afero.Fs
	TempFS     afero.Fs
}

type App struct {
	deps     Dependencies
	registry *registryruntime.Registry
	modules  []Module
	started  []Module
	mu       sync.RWMutex
}

func NewApp(deps Dependencies) *App {
	return &App{
		deps:     deps,
		registry: registryruntime.New(),
		modules:  make([]Module, 0),
		started:  make([]Module, 0),
	}
}

func (a *App) Dependencies() Dependencies {
	return a.deps
}

func (a *App) ModuleDeps() ModuleDeps {
	return ModuleDeps{
		Dependencies: a.deps,
		Registry:     a.registry,
	}
}

func (a *App) Registry() *registryruntime.Registry {
	return a.registry
}

func (a *App) RegisterModule(module Module) error {
	if module == nil {
		return errors.New("module is required")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	for _, existing := range a.modules {
		if existing.Name() == module.Name() {
			return fmt.Errorf("module %s already registered", module.Name())
		}
	}
	a.modules = append(a.modules, module)
	return nil
}

func (a *App) Modules() []Module {
	a.mu.RLock()
	defer a.mu.RUnlock()
	modules := make([]Module, len(a.modules))
	copy(modules, a.modules)
	return modules
}

func (a *App) RegisterRoutes(r chi.Router) {
	deps := a.ModuleDeps()
	for _, module := range a.Modules() {
		r.Group(func(r chi.Router) {
			r.Use(ContextMiddleware(deps))
			module.RegisterRoutes(r)
		})
	}
}

func (a *App) Start(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.started = a.started[:0]
	moduleDeps := a.ModuleDeps()
	for _, module := range a.modules {
		if err := module.Init(moduleDeps); err != nil {
			return fmt.Errorf("init module %s: %w", module.Name(), err)
		}
	}
	moduleCtx := moduleDeps.WithContext(ctx)
	for _, module := range a.modules {
		if err := module.Start(moduleCtx); err != nil {
			_ = a.stopStartedLocked(ctx)
			return fmt.Errorf("start module %s: %w", module.Name(), err)
		}
		a.started = append(a.started, module)
	}
	return nil
}

func (a *App) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.stopStartedLocked(ctx)
}

func (a *App) stopStartedLocked(ctx context.Context) error {
	var stopErr error
	moduleCtx := a.ModuleDeps().WithContext(ctx)
	for i := len(a.started) - 1; i >= 0; i-- {
		module := a.started[i]
		if err := module.Stop(moduleCtx); err != nil {
			stopErr = errors.Join(stopErr, fmt.Errorf("stop module %s: %w", module.Name(), err))
		}
	}
	a.started = a.started[:0]
	return stopErr
}
