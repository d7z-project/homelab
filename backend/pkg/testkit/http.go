package testkit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"homelab/pkg/common"
)

func (e *Env) ensureRouter() http.Handler {
	if e.Router != nil {
		return e.Router
	}
	e.Router = http.StripPrefix("/api/v1", e.App.Handler())
	return e.Router
}

func (e *Env) Do(method, path string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "testkit")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	e.ensureRouter().ServeHTTP(rec, req)
	return rec
}

func (e *Env) DoJSON(method, path, token string, body any) *httptest.ResponseRecorder {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			e.T.Fatalf("marshal request body: %v", err)
		}
	}

	headers := map[string]string{
		"Content-Type": "application/json",
	}
	if token != "" {
		headers["Authorization"] = "Bearer " + token
	}
	return e.Do(method, path, payload, headers)
}

func DecodeJSON[T any](t TestingT, rec *httptest.ResponseRecorder) T {
	t.Helper()

	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response body: %v; body=%s", err, rec.Body.String())
	}
	return out
}

func MustStatus(t TestingT, rec *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if rec.Code != expected {
		t.Fatalf("unexpected status: got %d want %d body=%s", rec.Code, expected, rec.Body.String())
	}
}

func RootToken(t TestingT, env *Env) string {
	t.Helper()

	rec := env.DoJSON(http.MethodPost, "/api/v1/auth/login", "", map[string]string{
		"password": common.Opts.RootPassword,
	})
	MustStatus(t, rec, http.StatusOK)
	body := DecodeJSON[struct {
		SessionID string `json:"session_id"`
	}](t, rec)
	if body.SessionID == "" {
		t.Fatalf("expected non-empty root token")
	}
	return body.SessionID
}
