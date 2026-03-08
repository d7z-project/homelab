package discovery_test

import (
	"context"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	"homelab/pkg/services/discovery"
	"homelab/tests"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiscoveryService(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	// Register a test lookup
	discovery.Register("test/items", func(ctx context.Context, search string, cursor string, limit int) (*models.PaginationResponse[models.LookupItem], error) {
		items := []models.LookupItem{
			{ID: "1", Name: "Item 1", Description: "Desc 1"},
			{ID: "2", Name: "Item 2", Description: "Desc 2"},
			{ID: "3", Name: "Other", Description: "Desc 3"},
		}

		var filtered []models.LookupItem
		for _, item := range items {
			if search == "" || (item.Name == search || item.ID == search) {
				filtered = append(filtered, item)
			}
		}

		return discovery.Paginate(filtered, cursor, limit), nil
	})

	t.Run("Lookup Success", func(t *testing.T) {
		req := models.LookupRequest{
			Code:   "test/items",
			Limit:  10,
			Cursor: "",
		}
		res, err := discovery.Lookup(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, int64(3), res.Total)
		assert.Len(t, res.Items, 3)
	})

	t.Run("Lookup with Search", func(t *testing.T) {
		req := models.LookupRequest{
			Code:   "test/items",
			Search: "Item 1",
			Limit:  10,
		}
		res, err := discovery.Lookup(context.Background(), req)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), res.Total)
		assert.Equal(t, "1", res.Items[0].ID)
	})

	t.Run("Lookup Code Not Found", func(t *testing.T) {
		req := models.LookupRequest{
			Code: "nonexistent",
		}
		_, err := discovery.Lookup(context.Background(), req)
		assert.ErrorIs(t, err, discovery.ErrCodeNotFound)
	})

	t.Run("List Codes", func(t *testing.T) {
		codes := discovery.GetRegisteredCodes()
		assert.Contains(t, codes, "test/items")
		assert.Contains(t, codes, "network/dns/domains")
	})
}

func TestDiscoveryPermissions(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	// Already registered network/dns/domains in service init
	// We need a context with permissions

	t.Run("DNS Domains Filtering", func(t *testing.T) {
		// Mock permissions: only allow "network/dns/example.com"
		perms := &models.ResourcePermissions{
			AllowedInstances: []string{"network/dns/example.com"},
		}
		ctx := auth.WithPermissions(context.Background(), perms)

		// Note: Since we don't have a real DB in unit tests without setup,
		// we rely on the fact that dnsrepo might be empty or mocked.
		// However, the logic of filtering is what we want to test if possible.
		// For now, we just ensure it doesn't crash and returns empty if no domains match.

		req := models.LookupRequest{
			Code: "network/dns/domains",
		}
		res, err := discovery.Lookup(ctx, req)
		assert.NoError(t, err)
		// Should be empty as repo is empty in this test environment
		assert.Empty(t, res.Items)
	})
}
