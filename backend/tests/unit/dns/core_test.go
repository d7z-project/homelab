package dns_test

import (
	"context"
	commonaudit "homelab/pkg/common/audit"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	auditservice "homelab/pkg/services/audit"
	dnsservice "homelab/pkg/services/dns"
	"homelab/tests"
	"testing"
	"time"
)

func TestDNSCacheInvalidationAndAudit(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// Inject AuditLogger into context for testing
	ctx := context.WithValue(context.Background(), commonaudit.LoggerContextKey, &commonaudit.AuditLogger{
		Subject:  "admin",
		Resource: "network/dns",
	})
	adminCtx := auth.WithPermissions(ctx, &models.ResourcePermissions{AllowedAll: true})

	// 1. Create a domain
	dom, _ := dnsservice.CreateDomain(adminCtx, &models.Domain{Name: "cache-test.com", Enabled: true})

	// 2. Initial Export (should populate cache)
	exp1, _ := dnsservice.ExportAll(adminCtx)

	// 3. Update domain (should invalidate cache and log changes)
	updateReq := *dom
	updateReq.Comments = "Updated Comment"
	_, err := dnsservice.UpdateDomain(adminCtx, dom.ID, &updateReq)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// 4. Verify Export changed (cache invalidation check)
	exp2, _ := dnsservice.ExportAll(adminCtx)
	if exp1 == exp2 {
		t.Error("Export cache was not invalidated after update")
	}

	// 5. Verify Audit Log was created
	time.Sleep(50 * time.Millisecond) // Wait for async audit log goroutine
	logs, _ := auditservice.ListLogs(adminCtx, 1, 10, "cache-test.com")
	foundUpdate := false
	if items, ok := logs.Items.([]interface{}); ok {
		for _, item := range items {
			l := item.(models.AuditLog)
			if l.Action == "UpdateDomain" {
				foundUpdate = true
				break
			}
		}
	}
	if !foundUpdate {
		t.Error("UpdateDomain audit log not found")
	}
}

func contains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestDNSFullWorkflow(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	perms := &models.ResourcePermissions{AllowedAll: true}
	ctx := auth.WithPermissions(context.Background(), perms)

	// Test Domain CRUD
	domainName := "example.com"
	domain, err := dnsservice.CreateDomain(ctx, &models.Domain{
		Name:     domainName,
		Enabled:  true,
		Comments: "Test Domain",
	})
	if err != nil {
		t.Fatalf("CreateDomain failed: %v", err)
	}
	if domain.ID == "" {
		t.Fatal("Expected Domain ID to be generated")
	}

	// Test Record CRUD
	record, err := dnsservice.CreateRecord(ctx, &models.Record{
		DomainID: domain.ID,
		Name:     "www",
		Type:     "A",
		Value:    "1.2.3.4",
		TTL:      600,
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}
	if record.ID == "" {
		t.Fatal("Expected Record ID to be generated")
	}

	// Test Update
	record.Value = "5.6.7.8"
	updated, err := dnsservice.UpdateRecord(ctx, record.ID, record)
	if err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}
	if updated.Value != "5.6.7.8" {
		t.Errorf("Expected updated value 5.6.7.8, got %s", updated.Value)
	}

	// Test List
	resp, err := dnsservice.ListRecords(ctx, domain.ID, 1, 10, "")
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	// Items includes SOA + A record
	if resp.Total < 2 {
		t.Errorf("Expected at least 2 records (SOA + A), got %d", resp.Total)
	}

	// Test Delete Record
	err = dnsservice.DeleteRecord(ctx, record.ID)
	if err != nil {
		t.Fatalf("DeleteRecord failed: %v", err)
	}

	// Test Delete Domain
	err = dnsservice.DeleteDomain(ctx, domain.ID)
	if err != nil {
		t.Fatalf("DeleteDomain failed: %v", err)
	}
}
