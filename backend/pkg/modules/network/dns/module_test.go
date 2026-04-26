package dns_test

import (
	"context"
	"testing"
	"time"

	commonauth "homelab/pkg/common/auth"
	discoverymodel "homelab/pkg/models/core/discovery"
	rbacmodel "homelab/pkg/models/core/rbac"
	dnsmodel "homelab/pkg/models/network/dns"
	moduledns "homelab/pkg/modules/network/dns"
	dnsrepo "homelab/pkg/repositories/network/dns"
	runtimepkg "homelab/pkg/runtime"
	"homelab/pkg/testkit"
)

func TestModuleStartRegistersDNSDiscoveryLookups(t *testing.T) {
	t.Parallel()

	env := testkit.StartApp(t,
		testkit.SeedModule("test.seed.dns", func(ctx context.Context, _ runtimepkg.ModuleDeps) error {
			now := time.Now()
			if err := dnsrepo.SaveDomain(ctx, &dnsmodel.Domain{
				ID: "domain-1",
				Meta: dnsmodel.DomainV1Meta{
					Name:      "example.com",
					Email:     "admin@example.com",
					PrimaryNS: "ns1.example.com.",
					Enabled:   true,
				},
				Status: dnsmodel.DomainV1Status{
					CreatedAt: now,
					UpdatedAt: now,
				},
				Generation: 1,
			}); err != nil {
				return err
			}
			return dnsrepo.SaveRecord(ctx, &dnsmodel.Record{
				ID: "record-1",
				Meta: dnsmodel.RecordV1Meta{
					DomainID: "domain-1",
					Name:     "www",
					Type:     "A",
					Value:    "10.0.0.10",
					TTL:      300,
					Enabled:  true,
				},
				Generation: 1,
			})
		}),
		moduledns.New(),
	)

	ctx := commonauth.WithPermissions(env.Context(), &rbacmodel.ResourcePermissions{AllowedAll: true})

	domains, err := env.Deps.Registry.Lookup(ctx, discoverymodel.LookupRequest{
		Code:  "network/dns/domains",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("lookup domains: %v", err)
	}
	if len(domains.Items) != 1 || domains.Items[0].Name != "example.com" {
		t.Fatalf("unexpected domain lookup result: %#v", domains.Items)
	}

	records, err := env.Deps.Registry.Lookup(ctx, discoverymodel.LookupRequest{
		Code:  "network/dns/records",
		Limit: 20,
	})
	if err != nil {
		t.Fatalf("lookup records: %v", err)
	}
	if len(records.Items) != 1 || records.Items[0].Name != "www (A) - example.com" {
		t.Fatalf("unexpected record lookup result: %#v", records.Items)
	}
}
