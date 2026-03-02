package unit

import (
	"context"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	dnsservice "homelab/pkg/services/dns"
	"homelab/tests"
	"testing"
)

func TestDNSFullWorkflow(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	perms := &models.ResourcePermissions{AllowedAll: true}
	ctx := auth.WithPermissions(context.Background(), perms)

	// 1. 创建域名
	domain, err := dnsservice.CreateDomain(ctx, &models.Domain{
		Name: "example.com",
	})
	if err != nil {
		t.Fatalf("CreateDomain failed: %v", err)
	}

	// 2. 创建 A 记录
	recordA, err := dnsservice.CreateRecord(ctx, &models.Record{
		DomainID: domain.ID,
		Name:     "www",
		Type:     "A",
		Value:    "1.2.3.4",
		TTL:      600,
	})
	if err != nil {
		t.Fatalf("Create A record failed: %v", err)
	}
	if recordA.ID == "" {
		t.Error("Expected record ID to be generated")
	}

	// 3. 校验重复 CNAME 冲突 (RFC 1034)
	_, err = dnsservice.CreateRecord(ctx, &models.Record{
		DomainID: domain.ID,
		Name:     "www",
		Type:     "CNAME",
		Value:    "other.com",
	})
	if err == nil {
		t.Error("Expected error when creating CNAME on existing A record, but got nil")
	}

	// 4. 校验非法 IP
	_, err = dnsservice.CreateRecord(ctx, &models.Record{
		DomainID: domain.ID,
		Name:     "mail",
		Type:     "A",
		Value:    "not-an-ip",
	})
	if err == nil {
		t.Error("Expected error for invalid IP, but got nil")
	}

	// 5. 列出记录并搜索
	resp, err := dnsservice.ListRecords(ctx, domain.ID, 1, 10, "www")
	if err != nil {
		t.Fatalf("ListRecords failed: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("Expected 1 record, got %d", resp.Total)
	}

	// 6. 删除域名 (验证级联删除)
	err = dnsservice.DeleteDomain(ctx, domain.ID)
	if err != nil {
		t.Fatalf("DeleteDomain failed: %v", err)
	}

	// 验证记录是否也被清理
	respAfter, _ := dnsservice.ListRecords(ctx, domain.ID, 1, 10, "")
	if respAfter != nil && respAfter.Total > 0 {
		t.Errorf("Expected 0 records after domain deletion, got %d", respAfter.Total)
	}
}

func TestCreateDomainPermissions(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// 模拟无权限
	ctxNoPerm := auth.WithPermissions(context.Background(), &models.ResourcePermissions{AllowedAll: false})
	_, err := dnsservice.CreateDomain(ctxNoPerm, &models.Domain{Name: "no-perm.com"})
	if err == nil {
		t.Error("Expected permission denied error, but got nil")
	}

	// 模拟特定实例权限
	ctxSpecific := auth.WithPermissions(context.Background(), &models.ResourcePermissions{
		AllowedInstances: []string{"dns/allowed.com"},
	})
	_, err = dnsservice.CreateDomain(ctxSpecific, &models.Domain{Name: "allowed.com"})
	if err != nil {
		t.Errorf("Expected allowed creation, but got error: %v", err)
	}
}
