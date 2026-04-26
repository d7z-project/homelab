package testkit

import (
	"context"
	"net/http"
	"testing"

	"homelab/pkg/common"
	auditrepo "homelab/pkg/repositories/core/audit"
	authrepo "homelab/pkg/repositories/core/auth"
	rbacrepo "homelab/pkg/repositories/core/rbac"
	secretrepo "homelab/pkg/repositories/core/secret"
	dnsrepo "homelab/pkg/repositories/network/dns"
	intrepo "homelab/pkg/repositories/network/intelligence"
	iprepo "homelab/pkg/repositories/network/ip"
	siterepo "homelab/pkg/repositories/network/site"
	actionrepo "homelab/pkg/repositories/workflow/actions"
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

	deps := runtimepkg.ModuleDeps{
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
	common.ConfigureInfrastructure(deps.DB, deps.Locker, deps.Subscriber)
	authrepo.Configure(deps.DB)
	secretrepo.Configure(deps.DB)
	rbacrepo.Configure(deps.DB)
	auditrepo.Configure(deps.DB)
	dnsrepo.Configure(deps.DB)
	intrepo.Configure(deps.DB)
	iprepo.Configure(deps.DB)
	siterepo.Configure(deps.DB)
	actionrepo.Configure(deps.DB)
	return deps
}

func NewApp(t *testing.T, modules ...runtimepkg.Module) *Env {
	t.Helper()

	deps := NewModuleDeps(t)
	common.ConfigureInfrastructure(deps.DB, deps.Locker, deps.Subscriber)
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
	return context.Background()
}

func SeedModule(name string, seed func(ctx context.Context, deps runtimepkg.ModuleDeps) error) runtimepkg.Module {
	return &seedModule{name: name, seed: seed}
}

type seedModule struct {
	name string
	deps runtimepkg.ModuleDeps
	seed func(ctx context.Context, deps runtimepkg.ModuleDeps) error
}

func (m *seedModule) Name() string { return m.name }
func (m *seedModule) Init(deps runtimepkg.ModuleDeps) error {
	m.deps = deps
	return nil
}
func (m *seedModule) Routes() runtimepkg.RouteHandler { return nil }
func (m *seedModule) Start(ctx context.Context) error { return m.seed(ctx, m.deps) }
func (m *seedModule) Stop(context.Context) error      { return nil }

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
