package dns

import (
	"context"
	"fmt"
	dnsmodel "homelab/pkg/models/network/dns"
	dnsrepo "homelab/pkg/repositories/network/dns"
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
				record := r
				if record.Status.SOA == nil {
					continue
				}
				record.Status.SOA.Serial = incrementSOASerial(record.Status.SOA.Serial)
				_ = dnsrepo.SaveRecord(ctx, &record)
				break
			}
		}
	}
}

func formatSOA(soa *dnsmodel.SOAStatus) string {
	if soa == nil {
		return ""
	}
	return fmt.Sprintf("%s %s %s %d %d %d %d", soa.MName, soa.RName, soa.Serial, soa.Refresh, soa.Retry, soa.Expire, soa.Minimum)
}

func recordValue(record *dnsmodel.Record) string {
	if record == nil {
		return ""
	}
	if record.Meta.Type == "SOA" {
		return formatSOA(record.Status.SOA)
	}
	return record.Meta.Value
}

func incrementSOASerial(serial string) string {
	today := time.Now().Format("20060102")
	if !strings.HasPrefix(serial, today) {
		return today + "01"
	}
	seq := 1
	fmt.Sscanf(serial[8:], "%d", &seq)
	return today + fmt.Sprintf("%02d", seq+1)
}
