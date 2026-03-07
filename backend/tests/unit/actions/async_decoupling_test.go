package actions_test

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/actions"
	"homelab/pkg/services/actions"
	_ "homelab/pkg/services/actions/processors" // 触发 init 注册
	"homelab/tests"
	"sync"
	"testing"
	"time"

	"gopkg.d7z.net/middleware/subscribe"
)

// MockSubscriber 模拟集群消息订阅器
type MockSubscriber struct {
	Messages chan string
}

func (m *MockSubscriber) Subscribe(ctx context.Context, topic string) (<-chan string, error) {
	return m.Messages, nil
}
func (m *MockSubscriber) Publish(ctx context.Context, topic string, message string) error {
	m.Messages <- message
	return nil
}
func (m *MockSubscriber) Child(topic ...string) subscribe.Subscriber { return m }
func (m *MockSubscriber) Close() error                               { return nil }

func TestActionsAsyncDecoupling(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	// 1. 初始化 Mock 环境
	mockSub := &MockSubscriber{Messages: make(chan string, 10)}
	common.Subscriber = mockSub
	defer func() { common.Subscriber = nil }()

	ctx := tests.SetupMockRootContext()

	// 准备一个简单的 Workflow
	wf := &models.Workflow{
		ID: "async-wf", Name: "Async Workflow", Enabled: true, ServiceAccountID: "sa",
		Steps: []models.Step{{ID: "s1", Type: "core/logger", Params: map[string]string{"message": "hello"}}},
	}
	_ = repo.SaveWorkflow(ctx, wf)

	t.Run("Trigger_Returns_Pending_Immediately", func(t *testing.T) {
		// 触发工作流
		instanceID, err := actions.TriggerWorkflow(ctx, wf, "root", "Manual", nil)
		if err != nil {
			t.Fatalf("TriggerWorkflow failed: %v", err)
		}

		// 立即检查数据库中的状态，必须是 Pending
		inst, _ := repo.GetTaskInstance(ctx, instanceID)
		if inst == nil {
			t.Fatal("Instance not found in DB")
		}
		if inst.Status != "Pending" {
			t.Errorf("Expected status Pending immediately after trigger, got %s", inst.Status)
		}

		// 检查消息队列是否收到了执行信号
		select {
		case msg := <-mockSub.Messages:
			if !startsWith(msg, "workflow_execute") {
				t.Errorf("Expected workflow_execute signal, got %s", msg)
			}
		case <-time.After(1 * time.Second):
			t.Fatal("Timeout waiting for cluster notification")
		}
	})

	t.Run("Execution_Starts_Only_After_Signal_Handled", func(t *testing.T) {
		// 再次触发一个新实例
		instanceID, _ := actions.TriggerWorkflow(ctx, wf, "root", "Manual", nil)

		// 模拟 TriggerManager 接收到信号并处理
		handlers := common.GetEventHandlers("workflow_execute")
		if len(handlers) == 0 {
			actions.GlobalTriggerManager.Start()
			handlers = common.GetEventHandlers("workflow_execute")
		}

		// 获取信号 payload
		msg := <-mockSub.Messages
		payload := msg[len("workflow_execute:"):]

		// 此时状态依然应该是 Pending
		inst, _ := repo.GetTaskInstance(ctx, instanceID)
		if inst.Status != "Pending" {
			t.Errorf("Expected status Pending before handling signal, got %s", inst.Status)
		}

		// 执行 Handler (模拟节点领任务)
		go handlers[0](ctx, payload)

		// 等待状态变为 Success
		success := false
		for i := 0; i < 50; i++ {
			inst, _ = repo.GetTaskInstance(ctx, instanceID)
			if inst.Status == "Success" {
				success = true
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		if !success {
			t.Errorf("Workflow did not complete after signal handling, current status: %s, error: %s", inst.Status, inst.Error)
		}
	})

	t.Run("Distributed_Lock_Prevents_Duplicate_Execution", func(t *testing.T) {
		instanceID, _ := actions.TriggerWorkflow(ctx, wf, "root", "Manual", nil)
		msg := <-mockSub.Messages
		payload := msg[len("workflow_execute:"):]

		handlers := common.GetEventHandlers("workflow_execute")

		var wg sync.WaitGroup
		wg.Add(2)

		start := make(chan struct{})
		for i := 0; i < 2; i++ {
			go func() {
				defer wg.Done()
				<-start
				handlers[0](ctx, payload)
			}()
		}

		close(start)
		wg.Wait()

		// 等待执行完成
		var inst *models.TaskInstance
		for i := 0; i < 50; i++ {
			inst, _ = repo.GetTaskInstance(ctx, instanceID)
			if inst.Status == "Success" || inst.Status == "Failed" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		if inst.Status != "Success" {
			t.Errorf("Workflow should have finished successfully, got %s", inst.Status)
		}
	})
}

func startsWith(s, substr string) bool {
	return len(s) >= len(substr) && s[0:len(substr)] == substr
}
