package common_test

import (
	"context"
	"errors"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/tests"
	"testing"
)

type DummyMeta struct {
	Name    string `json:"name"`
	Attr    int    `json:"attr"`
	FailVal bool   `json:"failVal"` // if true, ConfigValidator returns error
}

func (m DummyMeta) Validate(ctx context.Context) error {
	if m.FailVal {
		return errors.New("validation failed by request")
	}
	if m.Name == "" {
		return errors.New("name is required")
	}
	return nil
}

type DummyStatus struct {
	Active     bool   `json:"active"`
	LastUpdate string `json:"lastUpdate"`
}

func TestBaseRepository(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := context.Background()
	repo := common.NewBaseRepository[DummyMeta, DummyStatus]("test", "Dummy")

	t.Run("Create via Cow", func(t *testing.T) {
		err := repo.Cow(ctx, "id-1", func(res *models.Resource[DummyMeta, DummyStatus]) error {
			res.Meta.Name = "Test 1"
			res.Meta.Attr = 42
			res.Generation = 1
			res.ResourceVersion = 1
			return nil
		})
		if err != nil {
			t.Fatalf("Failed to create resource via Cow: %v", err)
		}

		res, err := repo.Get(ctx, "id-1")
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}
		if res.Meta.Name != "Test 1" || res.Meta.Attr != 42 {
			t.Errorf("Unexpected meta: %+v", res.Meta)
		}
	})

	t.Run("PatchMeta with ConfigValidator", func(t *testing.T) {
		err := repo.PatchMeta(ctx, "id-1", 1, func(m *DummyMeta) {
			m.Name = "Test 1 Updated"
		})
		if err != nil {
			t.Fatalf("Failed to patch meta: %v", err)
		}

		res, err := repo.Get(ctx, "id-1")
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}
		if res.Meta.Name != "Test 1 Updated" {
			t.Errorf("Unexpected meta: %+v", res.Meta)
		}
		if res.Generation != 2 {
			t.Errorf("Expected Generation to be 2, got %d", res.Generation)
		}
		if res.ResourceVersion != 2 {
			t.Errorf("Expected ResourceVersion to be 2, got %d", res.ResourceVersion)
		}
	})

	t.Run("PatchMeta with Invalid Config", func(t *testing.T) {
		err := repo.PatchMeta(ctx, "id-1", 2, func(m *DummyMeta) {
			m.FailVal = true
		})
		if err == nil {
			t.Fatalf("Expected ConfigValidator to fail, but it succeeded")
		}
		if !errors.Is(err, common.ErrInvalidConfig) {
			t.Errorf("Expected ErrInvalidConfig, got %v", err)
		}

		// Ensure generation didn't increment
		res, err := repo.Get(ctx, "id-1")
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}
		if res.Generation != 2 {
			t.Errorf("Expected Generation to remain 2, got %d", res.Generation)
		}
	})

	t.Run("PatchMeta with Conflict Generation", func(t *testing.T) {
		err := repo.PatchMeta(ctx, "id-1", 999, func(m *DummyMeta) {
			m.Name = "Should fail"
		})
		if err == nil {
			t.Fatalf("Expected generation conflict to fail, but it succeeded")
		}
		if !errors.Is(err, common.ErrConflict) {
			t.Errorf("Expected ErrConflict, got %v", err)
		}
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		err := repo.UpdateStatus(ctx, "id-1", func(s *DummyStatus) {
			s.Active = true
			s.LastUpdate = "now"
		})
		if err != nil {
			t.Fatalf("Failed to update status: %v", err)
		}

		res, err := repo.Get(ctx, "id-1")
		if err != nil {
			t.Fatalf("Failed to get resource: %v", err)
		}
		if !res.Status.Active || res.Status.LastUpdate != "now" {
			t.Errorf("Unexpected status: %+v", res.Status)
		}
		if res.Generation != 2 {
			t.Errorf("Expected Generation to remain 2, got %d", res.Generation)
		}
		if res.ResourceVersion != 3 {
			t.Errorf("Expected ResourceVersion to be 3, got %d", res.ResourceVersion)
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := repo.Delete(ctx, "id-1")
		if err != nil {
			t.Fatalf("Failed to delete resource: %v", err)
		}

		_, err = repo.Get(ctx, "id-1")
		if err == nil {
			t.Fatalf("Expected not found error, got nil")
		}
		if !errors.Is(err, common.ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("Get Non-existent", func(t *testing.T) {
		_, err := repo.Get(ctx, "non-existent")
		if err == nil {
			t.Fatalf("Expected not found error, got nil")
		}
		if !errors.Is(err, common.ErrNotFound) {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})
}
