package ip_test

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/ip"
	ipservice "homelab/pkg/services/ip"
	"homelab/tests"
	"testing"
	"time"

	"gopkg.d7z.net/middleware/subscribe"
)

// MockIPSubscriber 模拟集群订阅器
type MockIPSubscriber struct {
	Messages chan string
}

func (m *MockIPSubscriber) Subscribe(ctx context.Context, topic string) (<-chan string, error) {
	return m.Messages, nil
}
func (m *MockIPSubscriber) Publish(ctx context.Context, topic string, message string) error {
	m.Messages <- message
	return nil
}
func (m *MockIPSubscriber) Child(topic ...string) subscribe.Subscriber { return m }
func (m *MockIPSubscriber) Close() error                               { return nil }

func TestIPSyncAsyncDecoupling(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	// 1. 初始化 Mock 环境
	mockSub := &MockIPSubscriber{Messages: make(chan string, 10)}
	common.Subscriber = mockSub
	defer func() { common.Subscriber = nil }()

	// 初始化服务
	service := ipservice.NewIPPoolService(nil, nil)
	ctx := tests.SetupMockRootContext()

	// 2. 创建一个同步策略和目标池
	group := &models.IPGroup{ID: "test-group", Name: "Test Group"}
	_ = repo.SaveGroup(ctx, group)

	policy := &models.IPSyncPolicy{
		ID:            "test-policy",
		Name:          "Test Policy",
		Enabled:       true,
		SourceURL:     "http://localhost/ips.txt",
		Format:        "text",
		TargetGroupID: group.ID,
		Cron:          "0 0 * * *",
	}
	_ = repo.SaveSyncPolicy(ctx, policy)

	t.Run("IPSync_Trigger_Returns_Pending_Immediately", func(t *testing.T) {
		// 触发同步
		err := service.Sync(ctx, policy.ID)
		if err != nil {
			t.Fatalf("Sync trigger failed: %v", err)
		}

		// 验证状态立即变为 pending
		p, _ := repo.GetSyncPolicy(ctx, policy.ID)
		if p.LastStatus != models.TaskStatusPending {
			t.Errorf("Expected status pending, got %s", p.LastStatus)
		}

		// 验证信号已发出
		select {
		case msg := <-mockSub.Messages:
			if msg != "ip_sync_run:test-policy" {
				t.Errorf("Unexpected cluster message: %s", msg)
			}
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Timeout waiting for cluster signal")
		}
	})

	t.Run("IPSync_Execution_Is_Independent", func(t *testing.T) {
		// 模拟集群节点接收到信号并处理
		handlers := common.GetEventHandlers("ip_sync_run")
		if len(handlers) == 0 {
			t.Fatal("No handlers registered for ip_sync_run")
		}

		// 先触发以便建立 pending 状态和 Task 注册（模拟集群）
		_ = service.Sync(ctx, policy.ID)

		// 此时数据库状态应该是 pending
		p, _ := repo.GetSyncPolicy(ctx, policy.ID)
		if p.LastStatus != models.TaskStatusPending {
			t.Fatalf("Policy should be in pending state, got %s", p.LastStatus)
		}

		// 执行 Handler
		// 注意：doSync 内部会尝试下载，这里会因为 URL 无效而失败，
		// 但这恰恰能验证异步更新状态的逻辑（从 pending 变为 failed）
		handlers[0](context.Background(), policy.ID)

		// 稍微等等框架底层的 goroutine 和 Mutex 完成落盘 (框架包含网络拨号等)
		// 循环等待因为它是被发配到后台执行的 goroutine
		for i := 0; i < 20; i++ {
			p, _ = repo.GetSyncPolicy(ctx, policy.ID)
			if p.LastStatus != models.TaskStatusPending && p.LastStatus != models.TaskStatusRunning {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		// 验证状态已更新（异步完成）
		p, _ = repo.GetSyncPolicy(ctx, policy.ID)
		if p.LastStatus == models.TaskStatusPending {
			t.Error("Status should have changed from pending after execution")
		}

		// 验证最后运行时间已更新
		if p.LastRunAt.IsZero() {
			t.Error("LastRunAt should be set")
		}
	})
}
