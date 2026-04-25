package shared

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type testMetaBinder struct {
	bound bool
	err   error
}

func (m *testMetaBinder) Bind(_ *http.Request) error {
	m.bound = true
	return m.err
}

func TestResourceBindValidatesIDAndDelegatesToMetaBinder(t *testing.T) {
	t.Parallel()

	meta := testMetaBinder{}
	resource := Resource[testMetaBinder, struct{}]{
		ID:   "valid_id-1",
		Meta: meta,
	}

	req := httptest.NewRequest("POST", "/", nil)
	if err := resource.Bind(req); err != nil {
		t.Fatalf("bind resource: %v", err)
	}
	if !resource.Meta.bound {
		t.Fatal("expected meta binder to be called")
	}
}

func TestResourceBindRejectsInvalidID(t *testing.T) {
	t.Parallel()

	resource := Resource[testMetaBinder, struct{}]{
		ID: "INVALID ID",
	}

	err := resource.Bind(httptest.NewRequest("POST", "/", nil))
	if err == nil {
		t.Fatal("expected invalid id error")
	}
	if err.Error() != "invalid id format, only lowercase letters, numbers, underscores and hyphens are allowed" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResourceBindReturnsMetaBinderError(t *testing.T) {
	t.Parallel()

	expectedErr := errors.New("bind failed")
	resource := Resource[testMetaBinder, struct{}]{
		ID: "valid-id",
		Meta: testMetaBinder{
			err: expectedErr,
		},
	}

	err := resource.Bind(httptest.NewRequest("POST", "/", nil))
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected meta bind error, got %v", err)
	}
}
