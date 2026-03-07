package dns_test

import (
	"context"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsservice "homelab/pkg/services/dns"
	"homelab/tests"
	"strings"
	"testing"
)

func TestSOARecordLogic(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	perms := &models.ResourcePermissions{AllowedAll: true}
	ctx := auth.WithPermissions(context.Background(), perms)

	// 1. Create domain and check if SOA is automatically created
	domain, err := dnsservice.CreateDomain(ctx, &models.Domain{
		Name: "soa-test.com",
	})
	if err != nil {
		t.Fatalf("CreateDomain failed: %v", err)
	}

	resp, err := dnsservice.ListRecords(ctx, domain.ID, 1, 10, "")
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}

	var soaRecord *models.Record
	for _, item := range resp.Items.([]interface{}) {
		r := item.(models.Record)
		if r.Type == "SOA" {
			soaRecord = &r
			break
		}
	}

	if soaRecord == nil {
		t.Fatal("Expected SOA record to be created automatically, but not found")
	}

	if !strings.Contains(soaRecord.Value, "ns1.soa-test.com.") {
		t.Errorf("Expected default MNAME ns1.soa-test.com. in SOA, got %s", soaRecord.Value)
	}

	// 2. Try to create another SOA record (should fail)
	_, err = dnsservice.CreateRecord(ctx, &models.Record{
		DomainID: domain.ID,
		Name:     "@",
		Type:     "SOA",
		Value:    "ns2.other.com. admin.soa-test.com. 2026030301 7200 3600 1209600 3600",
	})
	if err == nil {
		t.Error("Expected error when manually creating SOA record, but got nil")
	}

	// 3. Try to delete SOA record (should fail)
	err = dnsservice.DeleteRecord(ctx, soaRecord.ID)
	if err == nil {
		t.Error("Expected error when deleting SOA record, but got nil")
	}

	// 4. Update SOA record (only MNAME and RNAME)
	updatedSOA := *soaRecord
	updatedSOA.Value = "ns2.new-master.com. new-admin.soa-test.com. 9999999999 1 1 1 1"
	res, err := dnsservice.UpdateRecord(ctx, soaRecord.ID, &updatedSOA)
	if err != nil {
		t.Fatalf("UpdateRecord (SOA) failed: %v", err)
	}

	if !strings.HasPrefix(res.Value, "ns2.new-master.com. new-admin.soa-test.com.") {
		t.Errorf("Expected updated MNAME and RNAME, got %s", res.Value)
	}
	if strings.Contains(res.Value, "9999999999") {
		t.Error("System-maintained SERIAL was overridden by user")
	}
	if strings.HasSuffix(res.Value, "1 1 1 1") {
		t.Error("System-maintained REFRESH/RETRY/EXPIRE/MINIMUM were overridden by user")
	}

	// 5. Create another record and check if SOA serial increments
	initialSerial := strings.Fields(res.Value)[2]
	_, err = dnsservice.CreateRecord(ctx, &models.Record{
		DomainID: domain.ID,
		Name:     "www",
		Type:     "A",
		Value:    "1.2.3.4",
	})
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// Re-fetch SOA
	resp2, _ := dnsservice.ListRecords(ctx, domain.ID, 1, 10, "")
	for _, item := range resp2.Items.([]interface{}) {
		r := item.(models.Record)
		if r.Type == "SOA" {
			newSerial := strings.Fields(r.Value)[2]
			if newSerial <= initialSerial {
				t.Errorf("Expected SOA serial to increment, but it didn't: %s -> %s", initialSerial, newSerial)
			}
			break
		}
	}
}
