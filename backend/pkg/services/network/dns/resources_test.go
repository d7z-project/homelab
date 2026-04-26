package dns

import "testing"

func TestDNSResourceBuilders(t *testing.T) {
	t.Parallel()

	if got := dnsResourceBase(); got != "network/dns" {
		t.Fatalf("unexpected dns resource base: %q", got)
	}

	if got := dnsDomainResource(" Example.COM "); got != "network/dns/domain/example.com" {
		t.Fatalf("unexpected dns domain resource: %q", got)
	}

	if got := dnsRecordResource(" Example.COM ", " * ", " cname "); got != "network/dns/domain/example.com/record/name/*/type/CNAME" {
		t.Fatalf("unexpected dns record resource: %q", got)
	}
}
