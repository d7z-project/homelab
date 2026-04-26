package common

import "testing"

func TestOptionsParseEnvBools(t *testing.T) {
	t.Setenv("HOMELAB_WORKFLOW", "false")
	t.Setenv("HOMELAB_INTELLIGENCE", "false")

	opts := &Options{
		Modules: ModuleOptions{
			Workflow:     true,
			Intelligence: true,
		},
	}
	opts.ParseEnv()

	if opts.Modules.Workflow {
		t.Fatal("expected workflow to be disabled from env")
	}
	if opts.Modules.Intelligence {
		t.Fatal("expected intelligence to be disabled from env")
	}
}
