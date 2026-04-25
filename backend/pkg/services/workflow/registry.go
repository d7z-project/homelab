package workflow

import (
	"sync"
)

var (
	registry = make(map[string]StepProcessor)
	mu       sync.RWMutex
)

func Register(processor StepProcessor) {
	mu.Lock()
	defer mu.Unlock()
	manifest := processor.Manifest()
	registry[manifest.ID] = processor
}

func GetProcessor(id string) (StepProcessor, bool) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[id]
	return p, ok
}

func ScanManifests() []StepManifest {
	mu.RLock()
	defer mu.RUnlock()
	res := make([]StepManifest, 0, len(registry))
	for _, p := range registry {
		res = append(res, p.Manifest())
	}
	return res
}
