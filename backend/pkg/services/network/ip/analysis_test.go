package ip_test

import (
	"context"
	"testing"

	runtimepkg "homelab/pkg/runtime"
	"homelab/pkg/services/network/ip"
)

func TestAnalysisEngineInfoWithoutEnricher(t *testing.T) {
	t.Parallel()

	engine := ip.NewAnalysisEngine(runtimepkg.ModuleDeps{}, nil)
	info, err := engine.Info(context.Background(), "1.1.1.1")
	if err != nil {
		t.Fatalf("info without enricher: %v", err)
	}
	if info == nil || info.IP != "1.1.1.1" {
		t.Fatalf("unexpected info: %#v", info)
	}
}
