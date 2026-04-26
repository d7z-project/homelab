package rules_test

import (
	"context"
	"testing"

	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	ipmodel "homelab/pkg/models/network/ip"
	sitemodel "homelab/pkg/models/network/site"
	iprepo "homelab/pkg/repositories/network/ip"
	siterepo "homelab/pkg/repositories/network/site"
	registryruntime "homelab/pkg/runtime/registry"
	ruleservice "homelab/pkg/services/rules"

	"gopkg.d7z.net/middleware/kv"
)

func TestRegisterDiscovery(t *testing.T) {
	t.Parallel()

	db, err := kv.NewKVFromURL("memory://")
	if err != nil {
		t.Fatalf("new memory kv: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})
	registry := registryruntime.New()
	iprepo.Configure(db)
	siterepo.Configure(db)
	ctx := context.Background()

	if err := iprepo.SavePool(ctx, &ipmodel.IPPool{
		ID:         "pool-1",
		Meta:       ipmodel.IPPoolV1Meta{Name: "cn-ip", Description: "china cidrs"},
		Generation: 1,
	}); err != nil {
		t.Fatalf("seed ip pool: %v", err)
	}
	if err := siterepo.SaveGroup(ctx, &sitemodel.SiteGroup{
		ID:         "group-1",
		Meta:       sitemodel.SiteGroupV1Meta{Name: "cn-site", Description: "china domains"},
		Generation: 1,
	}); err != nil {
		t.Fatalf("seed site group: %v", err)
	}

	ruleservice.RegisterDiscovery(registry)

	ctx = commonauth.WithPermissions(ctx, &rbacmodel.ResourcePermissions{AllowedAll: true})
	for _, tc := range []struct {
		code string
		name string
	}{
		{code: "network/ip/pools", name: "cn-ip"},
		{code: "network/site/pools", name: "cn-site"},
	} {
		lookup, err := registry.Lookup(ctx, discoverymodel.LookupRequest{Code: tc.code, Limit: 20})
		if err != nil {
			t.Fatalf("lookup %s: %v", tc.code, err)
		}
		if len(lookup.Items) != 1 || lookup.Items[0].Name != tc.name {
			t.Fatalf("unexpected lookup result for %s: %#v", tc.code, lookup.Items)
		}
	}

	for _, prefix := range []string{"network/ip", "network/site"} {
		suggestions, err := registry.SuggestResources(ctx, prefix)
		if err != nil {
			t.Fatalf("suggest resources %s: %v", prefix, err)
		}
		if len(suggestions) == 0 {
			t.Fatalf("expected suggestions for %s", prefix)
		}
	}
}
