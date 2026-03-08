package security_test

import (
	"fmt"
	"homelab/pkg/models"
	auditrepo "homelab/pkg/repositories/audit"
	auditservice "homelab/pkg/services/audit"
	"homelab/tests"
	"testing"
	"time"
)

func TestAuditLogsWorkflow(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := tests.SetupMockRootContext()

	// 1. 创建一批测试日志
	for i := 1; i <= 25; i++ {
		log := &models.AuditLog{
			Subject:   fmt.Sprintf("user-%02d", i),
			Action:    "Create",
			Resource:  "network/dns",
			TargetID:  fmt.Sprintf("domain-%d.com", i),
			Message:   "test message",
			Timestamp: time.Now().Add(time.Duration(i) * time.Second).Format(time.RFC3339),
		}
		if err := auditrepo.SaveLog(ctx, log); err != nil {
			t.Fatalf("Failed to save log %d: %v", i, err)
		}
	}

	// 2. 验证分页 (第一页)
	// 应返回最新的 10 条 (25, 24, ..., 16)
	res, err := auditservice.ScanLogs(ctx, "", 10, "")
	if err != nil {
		t.Fatalf("ScanLogs failed: %v", err)
	}
	if len(res.Items) != 10 {
		t.Errorf("Expected 10 items, got %d", len(res.Items))
	}

	// 验证顺序 (最新优先)
	firstLog := res.Items[0]
	if firstLog.Subject != "user-25" {
		t.Errorf("Expected first item subject 'user-25', got '%s'", firstLog.Subject)
	}

	// 3. 验证游标分页 (连续获取)
	// 第一页 10 条，第二页 10 条，第三页应剩 5 条
	res2, _ := auditservice.ScanLogs(ctx, res.NextCursor, 10, "")
	res3, _ := auditservice.ScanLogs(ctx, res2.NextCursor, 10, "")
	if len(res3.Items) != 5 {
		t.Errorf("Expected 5 items on the 3rd fetch, got %d", len(res3.Items))
	}

	// 4. 验证搜索
	resSearch, _ := auditservice.ScanLogs(ctx, "", 10, "user-05")
	if len(resSearch.Items) != 1 {
		t.Errorf("Expected 1 match for 'user-05', got %d", len(resSearch.Items))
	}
}
