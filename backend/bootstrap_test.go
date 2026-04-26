package main

import (
	"testing"

	intelligencemodel "homelab/pkg/models/network/intelligence"
	runtimepkg "homelab/pkg/runtime"
	"homelab/pkg/testkit"
)

func TestRegisterCoreModules(t *testing.T) {
	t.Parallel()

	deps := testkit.NewModuleDeps(t)

	app := runtimepkg.NewApp(deps.Dependencies)
	if err := registerModules(app, buildModules(deps, []intelligencemodel.IntelligenceSource{}, moduleOptions{
		enableWorkflow:     true,
		enableIntelligence: true,
	})); err != nil {
		t.Fatalf("register core modules: %v", err)
	}

	modules := app.Modules()
	if len(modules) != 11 {
		t.Fatalf("expected 11 modules, got %d", len(modules))
	}
	expectedNames := []string{
		"core.discovery",
		"core.auth",
		"core.session",
		"core.secret",
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

	deps := testkit.NewModuleDeps(t)

	modules := buildModules(deps, []intelligencemodel.IntelligenceSource{}, moduleOptions{
		enableWorkflow:     true,
		enableIntelligence: true,
	})
	if len(modules) != 11 {
		t.Fatalf("expected 11 modules, got %d", len(modules))
	}
}

func TestRegisterCoreModulesWithOptionalModulesDisabled(t *testing.T) {
	t.Parallel()

	deps := testkit.NewModuleDeps(t)

	app := runtimepkg.NewApp(deps.Dependencies)
	if err := registerModules(app, buildModules(deps, []intelligencemodel.IntelligenceSource{}, moduleOptions{})); err != nil {
		t.Fatalf("register core modules: %v", err)
	}

	modules := app.Modules()
	if len(modules) != 9 {
		t.Fatalf("expected 9 modules, got %d", len(modules))
	}
	expectedNames := []string{
		"core.discovery",
		"core.auth",
		"core.session",
		"core.secret",
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
