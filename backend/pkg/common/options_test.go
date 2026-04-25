package common

import "testing"

func TestOptionsParseEnvBools(t *testing.T) {
	t.Setenv("HOMELAB_WORKFLOW", "false")
	t.Setenv("HOMELAB_INTELLIGENCE", "false")

	opts := &Options{
		Workflow:     true,
		Intelligence: true,
	}
	opts.ParseEnv()

	if opts.Workflow {
		t.Fatal("expected workflow to be disabled from env")
	}
	if opts.Intelligence {
		t.Fatal("expected intelligence to be disabled from env")
	}
}
