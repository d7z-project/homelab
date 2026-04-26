package testkit

import (
	"context"
	"net/http"
	"testing"

	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"

	"github.com/spf13/afero"
	"gopkg.d7z.net/middleware/kv"
	"gopkg.d7z.net/middleware/lock"
	"gopkg.d7z.net/middleware/queue"
	"gopkg.d7z.net/middleware/subscribe"
)

type Env struct {
	T      *testing.T
	App    *runtimepkg.App
	Deps   runtimepkg.ModuleDeps
	Router http.Handler
}

func NewModuleDeps(t *testing.T) runtimepkg.ModuleDeps {
	t.Helper()

	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	locker, err := lock.NewLocker("memory://")
	if err != nil {
		t.Fatalf("new memory locker: %v", err)
	}
	registerClose(t, locker)

	taskQueue, err := queue.NewQueueFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory queue: %v", err)
	}
	registerClose(t, taskQueue)

	subscriber, err := subscribe.NewSubscriberFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory subscriber: %v", err)
	}
	registerClose(t, subscriber)

	return runtimepkg.ModuleDeps{
		Dependencies: runtimepkg.Dependencies{
			DB:         db,
			Locker:     locker,
			Queue:      taskQueue,
			Subscriber: subscriber,
			FS:         afero.NewMemMapFs(),
			TempFS:     afero.NewMemMapFs(),
		},
		Registry: registryruntime.New(),
	}
}

func NewApp(t *testing.T, modules ...runtimepkg.Module) *Env {
	t.Helper()

	deps := NewModuleDeps(t)
	app := runtimepkg.NewApp(deps.Dependencies)
	for _, module := range modules {
		if err := app.RegisterModule(module); err != nil {
			t.Fatalf("register module %s: %v", module.Name(), err)
		}
	}

	return &Env{
		T:   t,
		App: app,
		Deps: runtimepkg.ModuleDeps{
			Dependencies: deps.Dependencies,
			Registry:     app.Registry(),
		},
	}
}

func StartApp(t *testing.T, modules ...runtimepkg.Module) *Env {
	t.Helper()

	env := NewApp(t, modules...)
	if err := env.App.Start(env.Context()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	t.Cleanup(func() {
		if err := env.App.Stop(env.Context()); err != nil {
			t.Fatalf("stop app: %v", err)
		}
	})
	return env
}

func (e *Env) Context() context.Context {
	return e.Deps.WithContext(context.Background())
}

func SeedModule(name string, seed func(ctx context.Context, deps runtimepkg.ModuleDeps) error) runtimepkg.Module {
	return runtimepkg.FuncModule{
		ModuleName: name,
		OnStart: func(ctx context.Context) error {
			return seed(ctx, runtimepkg.MustModuleDeps(ctx))
		},
	}
}

func registerClose(t *testing.T, value any) {
	t.Helper()
	closer, ok := value.(interface{ Close() error })
	if !ok {
		return
	}
	t.Cleanup(func() {
		_ = closer.Close()
	})
}
