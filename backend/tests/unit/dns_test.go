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

func TestDNSExportFiltering(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	perms := &models.ResourcePermissions{AllowedAll: true}
	ctx := auth.WithPermissions(context.Background(), perms)

	// 1. 创建并禁用域名 (不应导出)
	inactiveDomain, err := dnsservice.CreateDomain(ctx, &models.Domain{
		Name:    "inactive-export.com",
		Enabled: false,
	})
	if err != nil {
		t.Fatalf("Failed to create inactive domain: %v", err)
	}
	_, err = dnsservice.CreateRecord(ctx, &models.Record{
		DomainID: inactiveDomain.ID,
		Name:     "www",
		Type:     "A",
		Value:    "1.1.1.1",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("Failed to create record for inactive domain: %v", err)
	}

	// 2. 创建激活域名及其记录
	activeDomain, err := dnsservice.CreateDomain(ctx, &models.Domain{
		Name:    "active-export.com",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("Failed to create active domain: %v", err)
	}
	// 激活记录 (应导出)
	_, err = dnsservice.CreateRecord(ctx, &models.Record{
		DomainID: activeDomain.ID,
		Name:     "www",
		Type:     "A",
		Value:    "2.2.2.2",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("Failed to create active record: %v", err)
	}
	// 禁用记录 (不应导出)
	_, err = dnsservice.CreateRecord(ctx, &models.Record{
		DomainID: activeDomain.ID,
		Name:     "api",
		Type:     "A",
		Value:    "3.3.3.3",
		Enabled:  false,
	})
	if err != nil {
		t.Fatalf("Failed to create inactive record: %v", err)
	}

	// 执行导出
	export, err := dnsservice.ExportAll(ctx)
	if err != nil {
		t.Fatalf("ExportAll failed: %v", err)
	}

	// 验证域名过滤
	foundActive := false
	for _, dom := range export.Domains {
		if dom.Name == "inactive-export.com" {
			t.Error("Export included inactive domain: inactive-export.com")
		}
		if dom.Name == "active-export.com" {
			foundActive = true
			// 验证记录过滤 (现在应包含 SOA 和 www A 记录)
			// dom.Records is map[name]map[type][]ExportRecord
			foundWWW := false
			foundSOA := false

			if types, ok := dom.Records["www"]; ok {
				if _, ok := types["A"]; ok {
					foundWWW = true
				}
			}
			if types, ok := dom.Records["@"]; ok {
				if _, ok := types["SOA"]; ok {
					foundSOA = true
				}
			}

			if !foundWWW {
				t.Error("Expected record www (A) not found in export")
			}
			if !foundSOA {
				t.Error("Expected SOA record (@) not found in export")
			}
		}
	}

	if !foundActive {
		t.Error("Export did not include active domain: active-export.com")
	}
}
