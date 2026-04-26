package runtime

import registryruntime "homelab/pkg/runtime/registry"

type ModuleDeps struct {
	Dependencies
	Registry *registryruntime.Registry
}
