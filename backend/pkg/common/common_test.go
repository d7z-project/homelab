package common

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"homelab/pkg/models/shared"
)

func TestCursorSuccessOmitsTotal(t *testing.T) {
	t.Parallel()

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/test", nil)

	CursorSuccess(rec, req, &shared.PaginationResponse[string]{
		Items:      []string{"a", "b"},
		NextCursor: "next",
		HasMore:    true,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var payload map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, exists := payload["total"]; exists {
		t.Fatalf("expected cursor response to omit total, got %s", rec.Body.String())
	}
	if payload["nextCursor"] != "next" {
		t.Fatalf("unexpected nextCursor: %#v", payload["nextCursor"])
	}
}
