package common

import (
	"context"
	"errors"
	"testing"

	"homelab/pkg/models/shared"

	"gopkg.d7z.net/middleware/kv"
)

type repoTestMeta struct {
	Value string `json:"value"`
}

func (m repoTestMeta) Validate(context.Context) error {
	if m.Value == "" {
		return errors.New("value is required")
	}
	return nil
}

type repoTestStatus struct {
	State string `json:"state"`
}

func newTestRepository(t *testing.T) (*ResourceRepository[repoTestMeta, repoTestStatus], context.Context) {
	t.Helper()
	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	return NewResourceRepository[repoTestMeta, repoTestStatus](db, "test", "objects"), context.Background()
}

func TestResourceRepositorySaveUpdateMetaAndStatus(t *testing.T) {
	t.Parallel()

	repo, ctx := newTestRepository(t)

	if err := repo.Save(ctx, &shared.Resource[repoTestMeta, repoTestStatus]{
		ID:     "alpha",
		Meta:   repoTestMeta{Value: "one"},
		Status: repoTestStatus{State: "new"},
	}); err != nil {
		t.Fatalf("save create: %v", err)
	}

	got, err := repo.Get(ctx, "alpha")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Meta.Value != "one" || got.Status.State != "new" {
		t.Fatalf("unexpected object: %#v", got)
	}
	if got.Generation != 1 || got.ResourceVersion != 1 {
		t.Fatalf("unexpected versions after create: %#v", got)
	}

	if err := repo.UpdateMeta(ctx, "alpha", 1, func(meta *repoTestMeta) {
		meta.Value = "two"
	}); err != nil {
		t.Fatalf("update meta: %v", err)
	}

	got, err = repo.Get(ctx, "alpha")
	if err != nil {
		t.Fatalf("get after patch: %v", err)
	}
	if got.Meta.Value != "two" {
		t.Fatalf("unexpected meta after patch: %#v", got)
	}
	if got.Generation != 2 || got.ResourceVersion != 2 {
		t.Fatalf("unexpected versions after patch: %#v", got)
	}

	if err := repo.UpdateStatus(ctx, "alpha", func(status *repoTestStatus) {
		status.State = "ready"
	}); err != nil {
		t.Fatalf("update status: %v", err)
	}

	got, err = repo.Get(ctx, "alpha")
	if err != nil {
		t.Fatalf("get after status update: %v", err)
	}
	if got.Status.State != "ready" {
		t.Fatalf("unexpected status after update: %#v", got)
	}
	if got.Generation != 2 || got.ResourceVersion != 3 {
		t.Fatalf("unexpected versions after status update: %#v", got)
	}
}

func TestResourceRepositoryListAndListAll(t *testing.T) {
	t.Parallel()

	repo, ctx := newTestRepository(t)

	for _, name := range []string{"alpha", "beta", "gamma"} {
		resourceName := name
		if err := repo.Save(ctx, &shared.Resource[repoTestMeta, repoTestStatus]{
			ID:   resourceName,
			Meta: repoTestMeta{Value: resourceName},
		}); err != nil {
			t.Fatalf("seed %s: %v", resourceName, err)
		}
	}

	first, err := repo.List(ctx, "", 2, nil)
	if err != nil {
		t.Fatalf("list first: %v", err)
	}
	if len(first.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(first.Items))
	}
	if first.NextCursor == "" || !first.HasMore {
		t.Fatalf("expected cursor continuation, got %#v", first)
	}

	second, err := repo.List(ctx, first.NextCursor, 2, nil)
	if err != nil {
		t.Fatalf("list second: %v", err)
	}
	if len(second.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(second.Items))
	}

	all, err := repo.ListAll(ctx)
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("expected 3 items, got %d", len(all))
	}
}
