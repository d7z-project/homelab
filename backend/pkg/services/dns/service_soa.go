package dns

import (
	"context"
	"errors"
	"fmt"
	dnsrepo "homelab/pkg/repositories/dns"
	"strings"
	"time"
)

const (
	defaultSOARefresh = 7200
	defaultSOARetry   = 3600
	defaultSOAExpire  = 1209600
	defaultSOAMinimum = 86400
)

func generateSOASerial() string { return time.Now().Format("20060102") + "01" }

func updateSOASerial(ctx context.Context, domainID string) {
	resp, _ := dnsrepo.ScanRecords(ctx, domainID, "", 100, "")
	if resp != nil {
		for _, r := range resp.Items {
			if r.Meta.Type == "SOA" {
				m, rn, _, _ := parseSOA(r.Meta.Value)
				r.Meta.Value = fmt.Sprintf("%s %s %s %d %d %d %d", m, rn, incrementSerial(r.Meta.Value), defaultSOARefresh, defaultSOARetry, defaultSOAExpire, defaultSOAMinimum)
				_ = dnsrepo.SaveRecord(ctx, &r)
				break
			}
		}
	}
}

func parseSOA(val string) (m, r, s string, err error) {
	p := strings.Fields(val)
	if len(p) < 3 {
		return "", "", "", errors.New("invalid SOA")
	}
	return p[0], p[1], p[2], nil
}

func incrementSerial(old string) string {
	_, _, s, err := parseSOA(old)
	today := time.Now().Format("20060102")
	if err != nil || !strings.HasPrefix(s, today) {
		return today + "01"
	}
	seq := 1
	fmt.Sscanf(s[8:], "%d", &seq)
	return today + fmt.Sprintf("%02d", seq+1)
}
