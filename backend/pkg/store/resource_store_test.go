package store_test

import (
	"context"
	"errors"
	"testing"

	metav1 "homelab/pkg/apis/meta/v1"
	"homelab/pkg/store"

	"gopkg.d7z.net/middleware/kv"
)

type testSpec struct {
	Value string `json:"value"`
}

type testStatus struct {
	State string `json:"state"`
}

func newTestStore(t *testing.T) *store.ResourceStore[testSpec, testStatus] {
	t.Helper()
	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return store.NewResourceStore[testSpec, testStatus](db, "test/v1", "TestObject", "testobjects", func(_ context.Context, spec *testSpec) error {
		if spec.Value == "" {
			return errors.New("value is required")
		}
		return nil
	})
}

func TestResourceStoreCreateGetAndDuplicate(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	obj := &metav1.Object[testSpec, testStatus]{
		Metadata: metav1.ObjectMeta{Name: "alpha"},
		Spec:     testSpec{Value: "one"},
		Status:   testStatus{State: "new"},
	}
	if err := s.Create(ctx, obj); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := s.Get(ctx, "alpha")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Metadata.Name != "alpha" {
		t.Fatalf("unexpected name: %s", got.Metadata.Name)
	}
	if got.Metadata.Generation != 1 {
		t.Fatalf("unexpected generation: %d", got.Metadata.Generation)
	}
	if got.Metadata.ResourceVersion != "1" {
		t.Fatalf("unexpected resourceVersion: %s", got.Metadata.ResourceVersion)
	}

	if err := s.Create(ctx, obj); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}
}

func TestResourceStoreUpdateSpecAndStatusVersioning(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	if err := s.Create(ctx, &metav1.Object[testSpec, testStatus]{
		Metadata: metav1.ObjectMeta{Name: "alpha"},
		Spec:     testSpec{Value: "one"},
		Status:   testStatus{State: "draft"},
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := s.UpdateSpec(ctx, "alpha", "1", func(spec *testSpec) error {
		spec.Value = "two"
		return nil
	}); err != nil {
		t.Fatalf("update spec: %v", err)
	}

	got, err := s.Get(ctx, "alpha")
	if err != nil {
		t.Fatalf("get after spec update: %v", err)
	}
	if got.Spec.Value != "two" {
		t.Fatalf("unexpected spec value: %s", got.Spec.Value)
	}
	if got.Metadata.Generation != 2 {
		t.Fatalf("expected generation 2, got %d", got.Metadata.Generation)
	}
	if got.Metadata.ResourceVersion != "2" {
		t.Fatalf("expected resourceVersion 2, got %s", got.Metadata.ResourceVersion)
	}

	if err := s.UpdateStatus(ctx, "alpha", "2", func(status *testStatus) error {
		status.State = "ready"
		return nil
	}); err != nil {
		t.Fatalf("update status: %v", err)
	}

	got, err = s.Get(ctx, "alpha")
	if err != nil {
		t.Fatalf("get after status update: %v", err)
	}
	if got.Status.State != "ready" {
		t.Fatalf("unexpected status state: %s", got.Status.State)
	}
	if got.Metadata.Generation != 2 {
		t.Fatalf("status update changed generation: %d", got.Metadata.Generation)
	}
	if got.Metadata.ResourceVersion != "3" {
		t.Fatalf("expected resourceVersion 3, got %s", got.Metadata.ResourceVersion)
	}
}

func TestResourceStoreConflictAndList(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	ctx := context.Background()

	for _, name := range []string{"alpha", "beta", "gamma"} {
		if err := s.Create(ctx, &metav1.Object[testSpec, testStatus]{
			Metadata: metav1.ObjectMeta{Name: name},
			Spec:     testSpec{Value: name},
		}); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	if err := s.UpdateSpec(ctx, "alpha", "99", func(spec *testSpec) error {
		spec.Value = "conflict"
		return nil
	}); !errors.Is(err, store.ErrConflict) {
		t.Fatalf("expected conflict, got %v", err)
	}

	first, err := s.List(ctx, "", 2, nil)
	if err != nil {
		t.Fatalf("list first page: %v", err)
	}
	if len(first.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(first.Items))
	}
	if first.Metadata.Continue == "" {
		t.Fatal("expected continue cursor")
	}

	second, err := s.List(ctx, first.Metadata.Continue, 2, nil)
	if err != nil {
		t.Fatalf("list second page: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(second.Items))
	}
}
