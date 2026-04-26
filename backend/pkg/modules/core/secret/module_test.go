package secret_test

import (
	"strings"
	"testing"

	"homelab/pkg/common"
	modulesecret "homelab/pkg/modules/core/secret"
	"homelab/pkg/testkit"
)

func TestModuleStartValidatesPlainSecretConfig(t *testing.T) {
	original := common.Opts.Secret
	common.Opts.Secret = "plain"
	t.Cleanup(func() {
		common.Opts.Secret = original
	})

	testkit.StartApp(t, modulesecret.New())
}

func TestModuleStartRejectsInvalidAES256Config(t *testing.T) {
	original := common.Opts.Secret
	common.Opts.Secret = "aes256:short"
	t.Cleanup(func() {
		common.Opts.Secret = original
	})

	env := testkit.NewApp(t, modulesecret.New())
	err := env.App.Start(env.Context())
	if err == nil || !strings.Contains(err.Error(), "32 bytes") {
		t.Fatalf("expected invalid aes256 config error, got %v", err)
	}
}
