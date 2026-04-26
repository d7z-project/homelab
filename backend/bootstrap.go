package main

import (
	intelligencemodel "homelab/pkg/models/network/intelligence"
	moduleaudit "homelab/pkg/modules/core/audit"
	moduleauth "homelab/pkg/modules/core/auth"
	modulediscovery "homelab/pkg/modules/core/discovery"
	modulerbac "homelab/pkg/modules/core/rbac"
	modulesecret "homelab/pkg/modules/core/secret"
	modulesession "homelab/pkg/modules/core/session"
	moduledns "homelab/pkg/modules/network/dns"
	moduleintelligence "homelab/pkg/modules/network/intelligence"
	moduleip "homelab/pkg/modules/network/ip"
	modulesite "homelab/pkg/modules/network/site"
	moduleworkflow "homelab/pkg/modules/workflow"
	runtimepkg "homelab/pkg/runtime"
	"homelab/pkg/services/network/ip"
)

type moduleOptions struct {
	enableWorkflow     bool
	enableIntelligence bool
}

func buildModules(deps runtimepkg.ModuleDeps, mmdbSources []intelligencemodel.IntelligenceSource, options moduleOptions) []runtimepkg.Module {
	var enricher *ip.MMDBManager
	if options.enableIntelligence {
		enricher = ip.NewMMDBManager(deps, mmdbSources)
	}
	modules := []runtimepkg.Module{
		modulediscovery.New(),
		moduleauth.New(),
		modulesession.New(),
		modulesecret.New(),
		modulerbac.New(),
		moduleaudit.New(),
		moduledns.New(),
		moduleip.New(enricher),
		modulesite.New(enricher),
	}

	if options.enableIntelligence {
		modules = append(modules, moduleintelligence.New(enricher))
	}
	if options.enableWorkflow {
		modules = append(modules, moduleworkflow.New())
	}

	return modules
}

func registerModules(app *runtimepkg.App, modules []runtimepkg.Module) error {
	for _, module := range modules {
		if err := app.RegisterModule(module); err != nil {
			return err
		}
	}
	return nil
}
