package dns_test

import (
	"context"
	commonaudit "homelab/pkg/common/audit"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsservice "homelab/pkg/services/dns"
	"homelab/tests"
	"testing"
)

func TestDNSCacheInvalidationAndAudit(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := context.WithValue(context.Background(), commonaudit.LoggerContextKey, &commonaudit.AuditLogger{
		Subject:  "admin",
		Resource: "network/dns",
	})
	adminCtx := auth.WithPermissions(ctx, &models.ResourcePermissions{AllowedAll: true})

	dom, _ := dnsservice.CreateDomain(adminCtx, &models.Domain{Meta: models.DomainV1Meta{Name: "cache-test.com", Enabled: true}})
	exp1, _ := dnsservice.ExportAll(adminCtx)

	updateReq := *dom
	updateReq.Meta.Enabled = false
	_, err := dnsservice.UpdateDomain(adminCtx, dom.ID, &updateReq)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	exp2, _ := dnsservice.ExportAll(adminCtx)
	if exp1 == exp2 {
		t.Error("Export cache was not invalidated")
	}
}

func TestDNSFullWorkflow(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	perms := &models.ResourcePermissions{AllowedAll: true}
	ctx := auth.WithPermissions(context.Background(), perms)

	domain, err := dnsservice.CreateDomain(ctx, &models.Domain{Meta: models.DomainV1Meta{Name: "example.com", Enabled: true}})
	if err != nil {
		t.Fatalf("CreateDomain failed: %v", err)
	}

	record, err := dnsservice.CreateRecord(ctx, &models.Record{Meta: models.RecordV1Meta{
		DomainID: domain.ID, Name: "www", Type: "A", Value: "1.2.3.4", TTL: 600, Enabled: true,
	}})
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	record.Meta.Value = "5.6.7.8"
	updated, err := dnsservice.UpdateRecord(ctx, record.ID, record)
	if err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}
	if updated.Meta.Value != "5.6.7.8" {
		t.Errorf("Expected 5.6.7.8")
	}

	_ = dnsservice.DeleteRecord(ctx, record.ID)
	_ = dnsservice.DeleteDomain(ctx, domain.ID)
}
