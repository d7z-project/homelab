package runtime_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	runtimepkg "homelab/pkg/runtime"

	"github.com/go-chi/chi/v5"
)

type testModule struct {
	name      string
	log       *[]string
	startErr  error
	stopErr   error
	routesHit *bool
}

func (m *testModule) Name() string { return m.name }

func (m *testModule) RegisterRoutes(r chi.Router) {
	if m.routesHit != nil {
		*m.routesHit = true
	}
	_ = r
}

func (m *testModule) Start(_ context.Context) error {
	*m.log = append(*m.log, "start:"+m.name)
	return m.startErr
}

func (m *testModule) Stop(_ context.Context) error {
	*m.log = append(*m.log, "stop:"+m.name)
	return m.stopErr
}

func TestAppStartStopOrder(t *testing.T) {
	t.Parallel()

	app := runtimepkg.NewApp(runtimepkg.Dependencies{})
	logs := make([]string, 0)
	routeA := false
	routeB := false

	if err := app.RegisterModule(&testModule{name: "a", log: &logs, routesHit: &routeA}); err != nil {
		t.Fatalf("register a: %v", err)
	}
	if err := app.RegisterModule(&testModule{name: "b", log: &logs, routesHit: &routeB}); err != nil {
		t.Fatalf("register b: %v", err)
	}

	app.RegisterRoutes(chi.NewRouter())
	if !routeA || !routeB {
		t.Fatal("expected routes to be registered for all modules")
	}

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start app: %v", err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatalf("stop app: %v", err)
	}

	expected := []string{"start:a", "start:b", "stop:b", "stop:a"}
	if !reflect.DeepEqual(logs, expected) {
		t.Fatalf("unexpected lifecycle order: %#v", logs)
	}
}

func TestAppRollsBackStartedModulesOnFailure(t *testing.T) {
	t.Parallel()

	app := runtimepkg.NewApp(runtimepkg.Dependencies{})
	logs := make([]string, 0)

	if err := app.RegisterModule(&testModule{name: "a", log: &logs}); err != nil {
		t.Fatalf("register a: %v", err)
	}
	if err := app.RegisterModule(&testModule{name: "b", log: &logs, startErr: errors.New("boom")}); err != nil {
		t.Fatalf("register b: %v", err)
	}

	if err := app.Start(context.Background()); err == nil {
		t.Fatal("expected start failure")
	}

	expected := []string{"start:a", "start:b", "stop:a"}
	if !reflect.DeepEqual(logs, expected) {
		t.Fatalf("unexpected rollback order: %#v", logs)
	}
}

func TestFuncModuleHooks(t *testing.T) {
	t.Parallel()

	logs := make([]string, 0)
	routeHit := false

	app := runtimepkg.NewApp(runtimepkg.Dependencies{})
	if err := app.RegisterModule(runtimepkg.FuncModule{
		ModuleName: "func",
		Routes: func(r chi.Router) {
			routeHit = true
			_ = r
		},
		OnStart: func(context.Context) error {
			logs = append(logs, "start")
			return nil
		},
		OnStop: func(context.Context) error {
			logs = append(logs, "stop")
			return nil
		},
	}); err != nil {
		t.Fatalf("register func module: %v", err)
	}

	app.RegisterRoutes(chi.NewRouter())
	if !routeHit {
		t.Fatal("expected routes hook to run")
	}
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("start func module app: %v", err)
	}
	if err := app.Stop(context.Background()); err != nil {
		t.Fatalf("stop func module app: %v", err)
	}

	expected := []string{"start", "stop"}
	if !reflect.DeepEqual(logs, expected) {
		t.Fatalf("unexpected func module lifecycle: %#v", logs)
	}
}
