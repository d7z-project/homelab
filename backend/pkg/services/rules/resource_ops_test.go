package rules

import (
	"context"
	"testing"
	"time"

	"homelab/pkg/common"
	"homelab/pkg/models"

	"gopkg.d7z.net/middleware/kv"
)

type testMeta struct {
	Name string `json:"name"`
}

type testStatus struct {
	CreatedAt time.Time `json:"createdAt"`
}

func newTestRepo(t *testing.T) ResourceRepository[testMeta, testStatus] {
	t.Helper()
	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	common.DB = db
	return common.NewBaseRepository[testMeta, testStatus]("rules", "tests")
}

func TestCreateAndLoadAndReplaceMeta(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	resource := &models.Resource[testMeta, testStatus]{
		ID:   "alpha",
		Meta: testMeta{Name: "Alpha"},
	}

	err := CreateAndLoad(context.Background(), repo, resource, func(res *models.Resource[testMeta, testStatus]) error {
		res.Meta = resource.Meta
		res.Status.CreatedAt = time.Unix(10, 0)
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	})
	if err != nil {
		t.Fatalf("create and load: %v", err)
	}
	if resource.Status.CreatedAt.IsZero() {
		t.Fatal("expected status to be loaded")
	}

	resource.Meta.Name = "Beta"
	if err := ReplaceMeta(context.Background(), repo, resource); err != nil {
		t.Fatalf("replace meta: %v", err)
	}

	got, err := repo.Get(context.Background(), "alpha")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Meta.Name != "Beta" {
		t.Fatalf("unexpected meta: %#v", got)
	}
}

func TestScanBySearch(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	for _, item := range []struct {
		id   string
		name string
	}{
		{id: "alpha", name: "First"},
		{id: "beta", name: "Second"},
		{id: "gamma", name: "Third"},
	} {
		resource := &models.Resource[testMeta, testStatus]{
			ID:   item.id,
			Meta: testMeta{Name: item.name},
		}
		if err := CreateAndLoad(context.Background(), repo, resource, func(res *models.Resource[testMeta, testStatus]) error {
			res.Meta = resource.Meta
			res.Generation = 1
			res.ResourceVersion = 1
			return nil
		}); err != nil {
			t.Fatalf("seed %s: %v", item.id, err)
		}
	}

	res, err := ScanBySearch(context.Background(), repo, "", 10, "sec", nil, func(meta *testMeta) string {
		return meta.Name
	})
	if err != nil {
		t.Fatalf("scan by search: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != "beta" {
		t.Fatalf("unexpected search result: %#v", res.Items)
	}
}
