package rules_test

import (
	"context"
	"testing"

	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
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
	common.DB = db

	if err := iprepo.PoolRepo.Cow(context.Background(), "pool-1", func(res *models.IPPool) error {
		res.ID = "pool-1"
		res.Meta = models.IPPoolV1Meta{Name: "cn-ip", Description: "china cidrs"}
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	}); err != nil {
		t.Fatalf("seed ip pool: %v", err)
	}
	if err := siterepo.GroupRepo.Cow(context.Background(), "group-1", func(res *models.SiteGroup) error {
		res.ID = "group-1"
		res.Meta = models.SiteGroupV1Meta{Name: "cn-site", Description: "china domains"}
		res.Generation = 1
		res.ResourceVersion = 1
		return nil
	}); err != nil {
		t.Fatalf("seed site group: %v", err)
	}

	ruleservice.RegisterDiscovery()

	ctx := commonauth.WithPermissions(context.Background(), &models.ResourcePermissions{AllowedAll: true})
	for _, tc := range []struct {
		code string
		name string
	}{
		{code: "network/ip/pools", name: "cn-ip"},
		{code: "network/site/pools", name: "cn-site"},
	} {
		lookup, err := registryruntime.Default().Lookup(ctx, models.LookupRequest{Code: tc.code, Limit: 20})
		if err != nil {
			t.Fatalf("lookup %s: %v", tc.code, err)
		}
		if len(lookup.Items) != 1 || lookup.Items[0].Name != tc.name {
			t.Fatalf("unexpected lookup result for %s: %#v", tc.code, lookup.Items)
		}
	}

	for _, prefix := range []string{"network/ip", "network/site"} {
		suggestions, err := registryruntime.Default().SuggestResources(ctx, prefix)
		if err != nil {
			t.Fatalf("suggest resources %s: %v", prefix, err)
		}
		if len(suggestions) == 0 {
			t.Fatalf("expected suggestions for %s", prefix)
		}
	}
}
