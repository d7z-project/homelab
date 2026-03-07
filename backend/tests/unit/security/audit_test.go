package security_test

import (
	"context"
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

	ctx := context.Background()

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
	resp, err := auditservice.ListLogs(ctx, 1, 10, "")
	if err != nil {
		t.Fatalf("ListLogs Page 1 failed: %v", err)
	}
	items := resp.Items.([]interface{})
	if len(items) != 10 {
		t.Errorf("Expected 10 items, got %d", len(items))
	}
	if resp.Total != 25 {
		t.Errorf("Expected total 25, got %d", resp.Total)
	}

	// 验证顺序 (最新优先)
	firstLog := items[0].(models.AuditLog)
	if firstLog.Subject != "user-25" {
		t.Errorf("Expected first item subject 'user-25', got '%s'", firstLog.Subject)
	}

	// 3. 验证分页 (第三页)
	// 应返回最后 5 条 (5, 4, 3, 2, 1)
	resp3, _ := auditservice.ListLogs(ctx, 3, 10, "")
	items3 := resp3.Items.([]interface{})
	if len(items3) != 5 {
		t.Errorf("Expected 5 items on page 3, got %d", len(items3))
	}

	// 4. 验证搜索
	respSearch, _ := auditservice.ListLogs(ctx, 1, 10, "user-05")
	if respSearch.Total != 1 {
		t.Errorf("Expected 1 match for 'user-05', got %d", respSearch.Total)
	}
}
