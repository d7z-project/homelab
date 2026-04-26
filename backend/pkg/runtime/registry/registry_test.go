package registry_test

import (
	"context"
	"errors"
	"testing"

	metav1 "homelab/pkg/apis/meta/v1"
	discoverymodel "homelab/pkg/models/core/discovery"
	"homelab/pkg/models/shared"
	"homelab/pkg/runtime/registry"
)

func TestRegistryRegisterAndList(t *testing.T) {
	t.Parallel()

	r := registry.New()
	err := r.RegisterResource(registry.ResourceDescriptor{
		Group:    "network",
		Resource: "dns",
		Kind:     "DNSZone",
		Verbs:    []string{"get", "list", "get"},
		DiscoverFunc: func(_ context.Context, _ string, _ string, _ int) (*metav1.List[discoverymodel.LookupItem], error) {
			return &metav1.List[discoverymodel.LookupItem]{Items: []discoverymodel.LookupItem{{ID: "zone-a", Name: "zone-a"}}}, nil
		},
	})
	if err != nil {
		t.Fatalf("register resource: %v", err)
	}

	if err := r.RegisterAction(registry.ActionDescriptor{
		ID:          "workflow.run",
		Category:    "workflow",
		Title:       "Run Workflow",
		Permissions: []string{"execute:actions"},
	}); err != nil {
		t.Fatalf("register action: %v", err)
	}

	resources := r.ListResources()
	if len(resources) != 1 {
		t.Fatalf("expected 1 resource, got %d", len(resources))
	}
	if len(resources[0].Verbs) != 2 {
		t.Fatalf("expected deduplicated verbs, got %v", resources[0].Verbs)
	}

	actions := r.ListActions()
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].ID != "workflow.run" {
		t.Fatalf("unexpected action id: %s", actions[0].ID)
	}
}

func TestRegistryRejectsDuplicates(t *testing.T) {
	t.Parallel()

	r := registry.New()
	if err := r.RegisterResource(registry.ResourceDescriptor{
		Group:    "network",
		Resource: "dns",
		Kind:     "DNSZone",
	}); err != nil {
		t.Fatalf("register resource: %v", err)
	}
	if err := r.RegisterResource(registry.ResourceDescriptor{
		Group:    "network",
		Resource: "dns",
		Kind:     "DNSZone",
	}); err == nil {
		t.Fatal("expected duplicate resource registration to fail")
	}

	if err := r.RegisterAction(registry.ActionDescriptor{ID: "workflow.run"}); err != nil {
		t.Fatalf("register action: %v", err)
	}
	if err := r.RegisterAction(registry.ActionDescriptor{ID: "workflow.run"}); err == nil {
		t.Fatal("expected duplicate action registration to fail")
	}
}

func TestRegistryLookupSuggestAndSAUsage(t *testing.T) {
	t.Parallel()

	r := registry.New()
	if err := r.RegisterLookup("network/dns/domains", func(_ context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		items := []discoverymodel.LookupItem{
			{ID: "example.com", Name: "example.com"},
			{ID: "internal.example", Name: "internal.example"},
		}
		filtered := make([]discoverymodel.LookupItem, 0, len(items))
		for _, item := range items {
			if search == "" || item.ID == search {
				filtered = append(filtered, item)
			}
		}
		return registry.Paginate(filtered, cursor, limit), nil
	}); err != nil {
		t.Fatalf("register lookup: %v", err)
	}

	if err := r.RegisterResource(registry.ResourceDescriptor{
		Group:    "network",
		Resource: "dns",
		Kind:     "DNSZone",
		Verbs:    []string{"get", "list"},
		DiscoverFunc: func(_ context.Context, prefix string, cursor string, limit int) (*metav1.List[discoverymodel.LookupItem], error) {
			if prefix != "" && prefix != "exa" {
				return &metav1.List[discoverymodel.LookupItem]{Items: []discoverymodel.LookupItem{}}, nil
			}
			return &metav1.List[discoverymodel.LookupItem]{
				Items: []discoverymodel.LookupItem{{ID: "example.com", Name: "example.com"}},
			}, nil
		},
	}); err != nil {
		t.Fatalf("register resource: %v", err)
	}

	found, err := r.Verify(context.Background(), "network/dns/domains", "example.com")
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !found {
		t.Fatal("expected lookup verification to find item")
	}

	suggestions, err := r.SuggestResources(context.Background(), "network/dns/")
	if err != nil {
		t.Fatalf("suggest resources: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("expected resource suggestions")
	}
	for _, item := range suggestions {
		if item.FullID == "network/dns/*" || item.FullID == "network/dns/**" {
			t.Fatalf("unexpected legacy wildcard suggestion: %#v", item)
		}
	}

	verbs, err := r.SuggestVerbs(context.Background(), "network/dns/domain/example.com")
	if err != nil {
		t.Fatalf("suggest verbs: %v", err)
	}
	if len(verbs) != 2 {
		t.Fatalf("expected 2 verbs, got %v", verbs)
	}

	r.RegisterSAUsageChecker(func(_ context.Context, id string) error {
		if id == "in-use" {
			return errors.New("in use")
		}
		return nil
	})
	if err := r.CheckSAUsage(context.Background(), "in-use"); err == nil {
		t.Fatal("expected SA usage check to fail")
	}
}

func TestRegistryScanCodesAndResourceSuggestions(t *testing.T) {
	t.Parallel()

	r := registry.New()
	if err := r.RegisterLookup("test/wrapper/lookup", func(_ context.Context, search string, cursor string, limit int) (*shared.PaginationResponse[discoverymodel.LookupItem], error) {
		items := []discoverymodel.LookupItem{
			{ID: "alpha", Name: "alpha"},
			{ID: "beta", Name: "beta"},
		}
		filtered := make([]discoverymodel.LookupItem, 0, len(items))
		for _, item := range items {
			if search == "" || item.ID == search {
				filtered = append(filtered, item)
			}
		}
		return registry.Paginate(filtered, cursor, limit), nil
	}); err != nil {
		t.Fatalf("register lookup: %v", err)
	}
	if err := r.RegisterResource(registry.ResourceDescriptor{
		Group:    "test/wrapper",
		Resource: "resource",
		Kind:     "test.wrapper.resource",
		Verbs:    []string{"get", "list"},
		DiscoverFunc: func(_ context.Context, prefix string, _ string, _ int) (*metav1.List[discoverymodel.LookupItem], error) {
			if prefix != "" && prefix != "a" {
				return &metav1.List[discoverymodel.LookupItem]{Items: []discoverymodel.LookupItem{}}, nil
			}
			return &metav1.List[discoverymodel.LookupItem]{
				Items: []discoverymodel.LookupItem{{ID: "alpha", Name: "alpha"}},
			}, nil
		},
	}); err != nil {
		t.Fatalf("register resource: %v", err)
	}

	res, err := r.Lookup(context.Background(), discoverymodel.LookupRequest{
		Code:  "test/wrapper/lookup",
		Limit: 1,
	})
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(res.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(res.Items))
	}

	codes := r.ScanCodes()
	if len(codes) != 1 || codes[0] != "test/wrapper/lookup" {
		t.Fatalf("unexpected codes: %v", codes)
	}

	suggestions, err := r.SuggestResources(context.Background(), "test/wrapper/resource/")
	if err != nil {
		t.Fatalf("suggest resources: %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("expected resource suggestions")
	}
	for _, item := range suggestions {
		if item.FullID == "test/wrapper/resource/*" || item.FullID == "test/wrapper/resource/**" {
			t.Fatalf("unexpected legacy wildcard suggestion: %#v", item)
		}
	}
}
