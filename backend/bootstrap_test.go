package main

import (
	"testing"

	"homelab/pkg/common"
	intelligencemodel "homelab/pkg/models/network/intelligence"
	runtimepkg "homelab/pkg/runtime"

	"github.com/spf13/afero"
	"gopkg.d7z.net/middleware/kv"
)

func setupBootstrapTestEnv(t *testing.T) {
	t.Helper()

	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	common.DB = db
	common.FS = afero.NewMemMapFs()
	common.TempDir = afero.NewMemMapFs()
}

func TestRegisterCoreModules(t *testing.T) {
	t.Parallel()

	setupBootstrapTestEnv(t)

	app := runtimepkg.NewApp(runtimepkg.Dependencies{})
	if err := registerModules(app, buildModules([]intelligencemodel.IntelligenceSource{}, moduleOptions{
		enableWorkflow:     true,
		enableIntelligence: true,
	})); err != nil {
		t.Fatalf("register core modules: %v", err)
	}

	modules := app.Modules()
	if len(modules) != 10 {
		t.Fatalf("expected 10 modules, got %d", len(modules))
	}
	expectedNames := []string{
		"core.discovery",
		"core.auth",
		"core.session",
		"core.rbac",
		"core.audit",
		"network.dns",
		"network.ip",
		"network.site",
		"network.intelligence",
		"workflow",
	}
	for i, module := range modules {
		if module.Name() != expectedNames[i] {
			t.Fatalf("unexpected module at %d: got %s want %s", i, module.Name(), expectedNames[i])
		}
	}
}

func TestBuildModules(t *testing.T) {
	t.Parallel()

	setupBootstrapTestEnv(t)

	modules := buildModules([]intelligencemodel.IntelligenceSource{}, moduleOptions{
		enableWorkflow:     true,
		enableIntelligence: true,
	})
	if len(modules) != 10 {
		t.Fatalf("expected 10 modules, got %d", len(modules))
	}
}

func TestRegisterCoreModulesWithOptionalModulesDisabled(t *testing.T) {
	t.Parallel()

	setupBootstrapTestEnv(t)

	app := runtimepkg.NewApp(runtimepkg.Dependencies{})
	if err := registerModules(app, buildModules([]intelligencemodel.IntelligenceSource{}, moduleOptions{})); err != nil {
		t.Fatalf("register core modules: %v", err)
	}

	modules := app.Modules()
	if len(modules) != 8 {
		t.Fatalf("expected 8 modules, got %d", len(modules))
	}
	expectedNames := []string{
		"core.discovery",
		"core.auth",
		"core.session",
		"core.rbac",
		"core.audit",
		"network.dns",
		"network.ip",
		"network.site",
	}
	for i, module := range modules {
		if module.Name() != expectedNames[i] {
			t.Fatalf("unexpected module at %d: got %s want %s", i, module.Name(), expectedNames[i])
		}
	}
}
