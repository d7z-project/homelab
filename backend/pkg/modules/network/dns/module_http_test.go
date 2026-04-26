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
	testkit.MustStatus(t, deniedCreate, http.StatusForbidden)

	allowedToken, err := testkit.SeedServiceAccount(env.Context(), "sa-dns-read", "dns read",
		rbacmodel.PolicyRule{Resource: "network/dns", Verbs: []string{"list"}},
	)
	if err != nil {
		t.Fatalf("seed allowed service account: %v", err)
	}
	allowedList := env.DoJSON(http.MethodGet, "/api/v1/network/dns/domains", allowedToken, nil)
	testkit.MustStatus(t, allowedList, http.StatusOK)
}

func TestDNSModuleInstanceScopedPermissions(t *testing.T) {
	env := testkit.StartApp(t, modulesecret.New(), moduleauth.New(), moduledns.New())
	rootToken := testkit.RootToken(t, env)

	createAllowedDomain := env.DoJSON(http.MethodPost, "/api/v1/network/dns/domains", rootToken, map[string]any{
		"meta": map[string]any{
			"name":      "allowed.example.com",
			"enabled":   true,
			"email":     "admin@allowed.example.com",
			"primaryNs": "ns1.allowed.example.com.",
		},
	})
	testkit.MustStatus(t, createAllowedDomain, http.StatusOK)
	allowedDomainBody := testkit.DecodeJSON[struct {
		ID string `json:"id"`
	}](t, createAllowedDomain)

	createBlockedDomain := env.DoJSON(http.MethodPost, "/api/v1/network/dns/domains", rootToken, map[string]any{
		"meta": map[string]any{
			"name":      "blocked.example.com",
			"enabled":   true,
			"email":     "admin@blocked.example.com",
			"primaryNs": "ns1.blocked.example.com.",
		},
	})
	testkit.MustStatus(t, createBlockedDomain, http.StatusOK)
	blockedDomainBody := testkit.DecodeJSON[struct {
		ID string `json:"id"`
	}](t, createBlockedDomain)

	instanceToken, err := testkit.SeedServiceAccount(env.Context(), "sa-dns-instance", "dns instance",
		rbacmodel.PolicyRule{Resource: "network/dns/domain/allowed.example.com", Verbs: []string{"list", "create"}},
		rbacmodel.PolicyRule{Resource: "network/dns/domain/allowed.example.com/record/name/www/type/A", Verbs: []string{"create"}},
	)
	if err != nil {
		t.Fatalf("seed instance service account: %v", err)
	}

	listResp := env.DoJSON(http.MethodGet, "/api/v1/network/dns/domains", instanceToken, nil)
	testkit.MustStatus(t, listResp, http.StatusOK)
	if got := listResp.Header().Get("X-Matched-Policy"); got != "network/dns/domain/allowed.example.com" {
		t.Fatalf("unexpected matched policy for instance list: %q", got)
	}
	listBody := testkit.DecodeJSON[struct {
		Items []struct {
			ID   string `json:"id"`
			Meta struct {
				Name string `json:"name"`
			} `json:"meta"`
		} `json:"items"`
	}](t, listResp)
	if len(listBody.Items) != 1 || listBody.Items[0].Meta.Name != "allowed.example.com" {
		t.Fatalf("expected only allowed domain, got %#v", listBody.Items)
	}

	allowedCreateDomain := env.DoJSON(http.MethodPost, "/api/v1/network/dns/domains", instanceToken, map[string]any{
		"meta": map[string]any{
			"name":      "allowed.example.com",
			"enabled":   true,
			"email":     "admin@allowed.example.com",
			"primaryNs": "ns1.allowed.example.com.",
		},
	})
	if allowedCreateDomain.Code == http.StatusForbidden {
		t.Fatalf("expected instance-scoped permission to pass route and service check, got forbidden: %s", allowedCreateDomain.Body.String())
	}
	if got := allowedCreateDomain.Header().Get("X-Matched-Policy"); got != "network/dns/domain/allowed.example.com" {
		t.Fatalf("expected matched policy for allowed domain create, got %q", got)
	}

	blockedCreateDomain := env.DoJSON(http.MethodPost, "/api/v1/network/dns/domains", instanceToken, map[string]any{
		"meta": map[string]any{
			"name":      "new-blocked.example.com",
			"enabled":   true,
			"email":     "admin@new-blocked.example.com",
			"primaryNs": "ns1.new-blocked.example.com.",
		},
	})
	testkit.MustStatus(t, blockedCreateDomain, http.StatusForbidden)
	if got := blockedCreateDomain.Header().Get("X-Matched-Policy"); got != "network/dns/domain/allowed.example.com" {
		t.Fatalf("expected route pass then service denial for mismatched domain create, got matched policy %q", got)
	}

	allowedCreateRecord := env.DoJSON(http.MethodPost, "/api/v1/network/dns/records", instanceToken, map[string]any{
		"meta": map[string]any{
			"domainId": allowedDomainBody.ID,
			"name":     "www",
			"type":     "A",
			"value":    "10.0.0.20",
			"ttl":      300,
			"enabled":  true,
		},
	})
	testkit.MustStatus(t, allowedCreateRecord, http.StatusOK)
	if got := allowedCreateRecord.Header().Get("X-Matched-Policy"); got != "network/dns/domain/allowed.example.com" {
		t.Fatalf("unexpected matched policy for allowed record create: %q", got)
	}

	blockedCreateRecord := env.DoJSON(http.MethodPost, "/api/v1/network/dns/records", instanceToken, map[string]any{
		"meta": map[string]any{
			"domainId": blockedDomainBody.ID,
			"name":     "api",
			"type":     "A",
			"value":    "10.0.0.21",
			"ttl":      300,
			"enabled":  true,
		},
	})
	testkit.MustStatus(t, blockedCreateRecord, http.StatusForbidden)
	if got := blockedCreateRecord.Header().Get("X-Matched-Policy"); got != "network/dns/domain/allowed.example.com" {
		t.Fatalf("expected route pass then service denial for record create, got matched policy %q", got)
	}
}

func TestDNSModuleWildcardRecordNamePermissions(t *testing.T) {
	env := testkit.StartApp(t, modulesecret.New(), moduleauth.New(), moduledns.New())
	rootToken := testkit.RootToken(t, env)

	createDomain := env.DoJSON(http.MethodPost, "/api/v1/network/dns/domains", rootToken, map[string]any{
		"meta": map[string]any{
			"name":      "wild.example.com",
			"enabled":   true,
			"email":     "admin@wild.example.com",
			"primaryNs": "ns1.wild.example.com.",
		},
	})
	testkit.MustStatus(t, createDomain, http.StatusOK)
	domainBody := testkit.DecodeJSON[struct {
		ID string `json:"id"`
	}](t, createDomain)

	wildcardToken, err := testkit.SeedServiceAccount(env.Context(), "sa-dns-wildcard-name", "dns wildcard name",
		rbacmodel.PolicyRule{Resource: "network/dns/domain/wild.example.com/record/name/*/type/CNAME", Verbs: []string{"create"}},
	)
	if err != nil {
		t.Fatalf("seed wildcard service account: %v", err)
	}

	createWildcardRecord := env.DoJSON(http.MethodPost, "/api/v1/network/dns/records", wildcardToken, map[string]any{
		"meta": map[string]any{
			"domainId": domainBody.ID,
			"name":     "*",
			"type":     "CNAME",
			"value":    "target.example.com.",
			"ttl":      300,
			"enabled":  true,
		},
	})
	testkit.MustStatus(t, createWildcardRecord, http.StatusOK)
	if got := createWildcardRecord.Header().Get("X-Matched-Policy"); got != "network/dns/domain/wild.example.com/record/name/*/type/CNAME" {
		t.Fatalf("unexpected matched policy for wildcard-name record: %q", got)
	}

	createDifferentRecord := env.DoJSON(http.MethodPost, "/api/v1/network/dns/records", wildcardToken, map[string]any{
		"meta": map[string]any{
			"domainId": domainBody.ID,
			"name":     "www",
			"type":     "CNAME",
			"value":    "target.example.com.",
			"ttl":      300,
			"enabled":  true,
		},
	})
	testkit.MustStatus(t, createDifferentRecord, http.StatusForbidden)
	if got := createDifferentRecord.Header().Get("X-Matched-Policy"); got != "network/dns/domain/wild.example.com/record/name/*/type/CNAME" {
		t.Fatalf("expected service denial with wildcard-name policy context, got %q", got)
	}
}
