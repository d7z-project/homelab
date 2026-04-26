package dns

import (
	"fmt"
	"strings"
)

const dnsResourceRoot = "network/dns"

func dnsResourceBase() string {
	return dnsResourceRoot
}

func dnsDomainResource(domain string) string {
	return fmt.Sprintf("%s/domain/%s", dnsResourceRoot, normalizeDNSDomainResourcePart(domain))
}

func dnsRecordResource(domain, name, recordType string) string {
	return fmt.Sprintf(
		"%s/record/name/%s/type/%s",
		dnsDomainResource(domain),
		normalizeDNSRecordNameResourcePart(name),
		normalizeDNSRecordTypeResourcePart(recordType),
	)
}

func normalizeDNSDomainResourcePart(domain string) string {
	return strings.ToLower(strings.TrimSpace(domain))
}

func normalizeDNSRecordNameResourcePart(name string) string {
	return strings.TrimSpace(name)
}

func normalizeDNSRecordTypeResourcePart(recordType string) string {
	return strings.ToUpper(strings.TrimSpace(recordType))
}
