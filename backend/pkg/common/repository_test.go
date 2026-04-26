package common

import (
	"context"
	"errors"
	"testing"

	"homelab/pkg/models/shared"
	runtimepkg "homelab/pkg/runtime"
	registryruntime "homelab/pkg/runtime/registry"

	"github.com/spf13/afero"
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

func newTestRepository(t *testing.T) (*BaseRepository[repoTestMeta, repoTestStatus], context.Context) {
	t.Helper()
	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	deps := runtimepkg.ModuleDeps{
		Dependencies: runtimepkg.Dependencies{
			DB:     db,
			FS:     afero.NewMemMapFs(),
			TempFS: afero.NewMemMapFs(),
		},
		Registry: registryruntime.New(),
	}
	return NewBaseRepository[repoTestMeta, repoTestStatus]("test", "objects"), deps.WithContext(context.Background())
}

func TestBaseRepositoryCowPatchMetaAndStatus(t *testing.T) {
	t.Parallel()

	repo, ctx := newTestRepository(t)

	if err := repo.Cow(ctx, "alpha", func(res *shared.Resource[repoTestMeta, repoTestStatus]) error {
		res.ID = "alpha"
		res.Meta = repoTestMeta{Value: "one"}
		res.Status = repoTestStatus{State: "new"}
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	}); err != nil {
		t.Fatalf("cow create: %v", err)
	}

	got, err := repo.Get(ctx, "alpha")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Meta.Value != "one" || got.Status.State != "new" {
		t.Fatalf("unexpected object: %#v", got)
	}

	if err := repo.PatchMeta(ctx, "alpha", 1, func(meta *repoTestMeta) {
		meta.Value = "two"
	}); err != nil {
		t.Fatalf("patch meta: %v", err)
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

func TestBaseRepositoryListAndListAll(t *testing.T) {
	t.Parallel()

	repo, ctx := newTestRepository(t)

	for _, name := range []string{"alpha", "beta", "gamma"} {
		resourceName := name
		if err := repo.Cow(ctx, resourceName, func(res *shared.Resource[repoTestMeta, repoTestStatus]) error {
			res.ID = resourceName
			res.Meta = repoTestMeta{Value: resourceName}
			res.Generation = 1
			res.ResourceVersion = 1
			return nil
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
