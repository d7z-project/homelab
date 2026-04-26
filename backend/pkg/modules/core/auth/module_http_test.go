package auth_test

import (
	"net/http"
	"testing"

	moduleauth "homelab/pkg/modules/core/auth"
	modulesession "homelab/pkg/modules/core/session"
	"homelab/pkg/testkit"
)

func TestAuthModuleLoginInfoLogoutFlow(t *testing.T) {
	env := testkit.StartApp(t, moduleauth.New(), modulesession.New())

	token := testkit.RootToken(t, env)

	info := env.DoJSON(http.MethodGet, "/api/v1/auth/info", token, nil)
	testkit.MustStatus(t, info, http.StatusOK)
	body := testkit.DecodeJSON[struct {
		Type string `json:"type"`
	}](t, info)
	if body.Type != "root" {
		t.Fatalf("unexpected auth info type: %q", body.Type)
	}

	sessions := env.DoJSON(http.MethodGet, "/api/v1/auth/sessions", token, nil)
	testkit.MustStatus(t, sessions, http.StatusOK)

	logout := env.DoJSON(http.MethodPost, "/api/v1/auth/logout", token, nil)
	testkit.MustStatus(t, logout, http.StatusOK)

	infoAfterLogout := env.DoJSON(http.MethodGet, "/api/v1/auth/info", token, nil)
	testkit.MustStatus(t, infoAfterLogout, http.StatusUnauthorized)
}

func TestAuthModuleRejectsUnauthorizedInfo(t *testing.T) {
	env := testkit.StartApp(t, moduleauth.New())

	rec := env.DoJSON(http.MethodGet, "/api/v1/auth/info", "", nil)
	testkit.MustStatus(t, rec, http.StatusUnauthorized)
}
