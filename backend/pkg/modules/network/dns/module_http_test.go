package dns_test

import (
	"net/http"
	"testing"

	rbacmodel "homelab/pkg/models/core/rbac"
	moduleauth "homelab/pkg/modules/core/auth"
	modulesecret "homelab/pkg/modules/core/secret"
	moduledns "homelab/pkg/modules/network/dns"
	dnsrepo "homelab/pkg/repositories/network/dns"
	"homelab/pkg/testkit"
)

func TestDNSModuleCRUDAndPermissions(t *testing.T) {
	env := testkit.StartApp(t, modulesecret.New(), moduleauth.New(), moduledns.New())
	rootToken := testkit.RootToken(t, env)

	createDomain := env.DoJSON(http.MethodPost, "/api/v1/network/dns/domains", rootToken, map[string]any{
		"meta": map[string]any{
			"name":      "example.com",
			"enabled":   true,
			"email":     "admin@example.com",
			"primaryNs": "ns1.example.com.",
		},
	})
	testkit.MustStatus(t, createDomain, http.StatusOK)
	domainBody := testkit.DecodeJSON[struct {
		ID string `json:"id"`
	}](t, createDomain)
	if domainBody.ID == "" {
		t.Fatalf("expected created domain id")
	}

	createRecord := env.DoJSON(http.MethodPost, "/api/v1/network/dns/records", rootToken, map[string]any{
		"meta": map[string]any{
			"domainId": domainBody.ID,
			"name":     "www",
			"type":     "A",
			"value":    "10.0.0.10",
			"ttl":      300,
			"enabled":  true,
		},
	})
	testkit.MustStatus(t, createRecord, http.StatusOK)

	updateDomain := env.DoJSON(http.MethodPut, "/api/v1/network/dns/domains/"+domainBody.ID, rootToken, map[string]any{
		"id": domainBody.ID,
		"meta": map[string]any{
			"name":      "example.com",
			"enabled":   true,
			"email":     "hostmaster@example.com",
			"primaryNs": "ns2.example.com.",
		},
	})
	testkit.MustStatus(t, updateDomain, http.StatusOK)

	records, err := dnsrepo.ScanAllRecords(env.Context())
	if err != nil {
		t.Fatalf("scan records: %v", err)
	}
	foundSOA := false
	for _, record := range records {
		if record.Meta.DomainID == domainBody.ID && record.Meta.Type == "SOA" && record.Status.SOA != nil {
			foundSOA = true
			if record.Status.SOA.MName != "ns2.example.com." || record.Status.SOA.RName != "hostmaster.example.com." {
				t.Fatalf("unexpected soa values: %#v", record.Status.SOA)
			}
		}
	}
	if !foundSOA {
		t.Fatalf("expected system SOA record")
	}

	deniedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-dns-denied", "dns denied")
	if err != nil {
		t.Fatalf("seed denied service account: %v", err)
	}
	deniedCreate := env.DoJSON(http.MethodPost, "/api/v1/network/dns/domains", deniedToken, map[string]any{
		"meta": map[string]any{
			"name":    "denied.example.com",
			"enabled": true,
		},
	})
	testkit.MustStatus(t, deniedCreate, http.StatusUnauthorized)

	allowedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-dns-read", "dns read",
		rbacmodel.PolicyRule{Resource: "network/dns", Verbs: []string{"list"}},
	)
	if err != nil {
		t.Fatalf("seed allowed service account: %v", err)
	}
	allowedList := env.DoJSON(http.MethodGet, "/api/v1/network/dns/domains", allowedToken, nil)
	testkit.MustStatus(t, allowedList, http.StatusOK)
}
