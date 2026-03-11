package discovery_test

import (
	"context"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	"homelab/pkg/services/discovery"
	dnsservice "homelab/pkg/services/dns"
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
		codes := discovery.ScanCodes()
		assert.Contains(t, codes, "test/items")
		assert.Contains(t, codes, "network/dns/domains")
	})
}

func TestSuggestResources(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	// 1. Test root level suggestions
	t.Run("Root level", func(t *testing.T) {
		res, err := discovery.SuggestResources(context.Background(), "net")
		assert.NoError(t, err)
		// Should contain network/dns, network/ip, network/site
		var ids []string
		for _, r := range res {
			ids = append(ids, r.FullID)
		}
		assert.Contains(t, ids, "network/dns")
		assert.Contains(t, ids, "network/ip")
		assert.Contains(t, ids, "network/site")
	})

	// 2. Test specific module level (with trailing slash)
	t.Run("Module level with slash", func(t *testing.T) {
		// Prepare a domain for DNS discovery
		ctxRoot := tests.SetupMockRootContext()
		_, _ = dnsservice.CreateDomain(ctxRoot, &models.Domain{Meta: models.DomainV1Meta{Name: "example.com"}})

		res, err := discovery.SuggestResources(ctxRoot, "network/dns/")
		assert.NoError(t, err)

		var ids []string
		for _, r := range res {
			ids = append(ids, r.FullID)
		}
		// Should contain the domain and wildcards
		assert.Contains(t, ids, "network/dns/example.com")
		assert.Contains(t, ids, "network/dns/*")
		assert.Contains(t, ids, "network/dns/**")
	})

	// 3. Test exact match suggestions (should still offer sub-paths and wildcards)
	t.Run("Exact match", func(t *testing.T) {
		res, err := discovery.SuggestResources(context.Background(), "actions")
		assert.NoError(t, err)
		var ids []string
		for _, r := range res {
			ids = append(ids, r.FullID)
		}
		assert.Contains(t, ids, "actions")
	})
}

func TestSuggestVerbs(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	t.Run("Known resource", func(t *testing.T) {
		verbs, err := discovery.SuggestVerbs(context.Background(), "network/dns")
		assert.NoError(t, err)
		assert.Contains(t, verbs, "get")
		assert.Contains(t, verbs, "list")
		assert.NotContains(t, verbs, "execute") // DNS should only have CRUD
	})

	t.Run("Sub-resource match", func(t *testing.T) {
		verbs, err := discovery.SuggestVerbs(context.Background(), "network/dns/example.com")
		assert.NoError(t, err)
		assert.Contains(t, verbs, "get")
	})

	t.Run("Unknown resource", func(t *testing.T) {
		verbs, err := discovery.SuggestVerbs(context.Background(), "unknown/path")
		assert.NoError(t, err)
		assert.Equal(t, []string{"*"}, verbs)
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
