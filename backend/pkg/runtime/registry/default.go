package registry

var defaultRegistry = New()

func Default() *Registry {
	return defaultRegistry
}
