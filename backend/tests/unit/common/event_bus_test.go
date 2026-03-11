package common_test

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
	_ "homelab/pkg/services/actions/processors"
	"homelab/pkg/services/rbac"
	"homelab/tests"
	"sync/atomic"
	"testing"
	"time"
)

func TestEventBusDispatch(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	var received int32

	common.RegisterEventHandler("test_event", func(ctx context.Context, payload string) {
		if payload == "hello" {
			atomic.AddInt32(&received, 1)
		}
	})

	// Simulate Pub/Sub by directly calling NotifyCluster
	// In test environment, Subscriber is nil, so we test the handler registry directly
	t.Run("Handler Registration and Dispatch", func(t *testing.T) {
		// 直接触发 Handler 分发逻辑 (模拟集群接收)
		ctx := context.Background()
		common.TriggerEvent(ctx, "test_event", "hello")

		count := atomic.LoadInt32(&received)
		if count != 1 {
			t.Errorf("Expected handler to be called 1 time, got %d", count)
		}
	})

	t.Run("Multiple Handlers Same Event", func(t *testing.T) {
		var count2 int32
		common.RegisterEventHandler("multi_event", func(ctx context.Context, payload string) {
			atomic.AddInt32(&count2, 1)
		})
		common.RegisterEventHandler("multi_event", func(ctx context.Context, payload string) {
			atomic.AddInt32(&count2, 10)
		})

		common.TriggerEvent(context.Background(), "multi_event", "data")

		c := atomic.LoadInt32(&count2)
		if c != 11 {
			t.Errorf("Expected both handlers called (sum=11), got %d", c)
		}
	})

	t.Run("Unknown Event No Panic", func(t *testing.T) {
		common.TriggerEvent(context.Background(), "nonexistent_event", "whatever")
	})
}

func TriggerEvent(event string, payload string) {
	common.TriggerEvent(context.Background(), event, payload)
}

func TestGlobalVersionTracking(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()
	ctx := context.Background()

	t.Run("UpdateAndGetVersion", func(t *testing.T) {
		v1 := common.UpdateGlobalVersion(ctx, "test/module")
		if v1 <= 0 {
			t.Errorf("Expected positive version, got %d", v1)
		}

		retrieved := common.GetGlobalVersion(ctx, "test/module")
		if retrieved != v1 {
			t.Errorf("Expected version %d, got %d", v1, retrieved)
		}
	})

	t.Run("VersionIncreases", func(t *testing.T) {
		v1 := common.UpdateGlobalVersion(ctx, "test/incr")
		time.Sleep(1 * time.Millisecond) // ensure different UnixNano
		v2 := common.UpdateGlobalVersion(ctx, "test/incr")
		if v2 <= v1 {
			t.Errorf("Expected v2 (%d) > v1 (%d)", v2, v1)
		}
	})

	t.Run("DifferentModulesIndependent", func(t *testing.T) {
		common.UpdateGlobalVersion(ctx, "mod/a")
		common.UpdateGlobalVersion(ctx, "mod/b")

		va := common.GetGlobalVersion(ctx, "mod/a")
		vb := common.GetGlobalVersion(ctx, "mod/b")

		if va == 0 || vb == 0 {
			t.Errorf("Both modules should have versions, got a=%d b=%d", va, vb)
		}
	})

	t.Run("NonExistentModuleReturnsZero", func(t *testing.T) {
		v := common.GetGlobalVersion(ctx, "does/not/exist")
		if v != 0 {
			t.Errorf("Expected 0 for non-existent module, got %d", v)
		}
	})
}

func TestWorkflowTriggerClusterSync(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := tests.SetupMockRootContext()
	_, _ = rbac.CreateServiceAccount(ctx, &models.ServiceAccount{ID: "sa_trigger", Meta: models.ServiceAccountV1Meta{Name: "SA Trigger"}})

	// Register mock processor
	actions.Register(&tests.MockProcessor{})

	t.Run("CreateWorkflowRegistersLocalCron", func(t *testing.T) {
		wf := &models.Workflow{ID: "s1", Meta: models.WorkflowV1Meta{
			Name:             "cron-sync-test",
			Enabled:          true,
			CronEnabled:      true,
			CronExpr:         "@every 1h",
			ServiceAccountID: "sa_trigger",
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock", Params: map[string]string{}},
			},
		}}

		created, err := actions.CreateWorkflow(ctx, wf)
		if err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		if created.ID == "" {
			t.Fatal("Expected workflow ID to be set")
		}

		// Verify the cron job was registered
		// We can verify indirectly by updating and checking no error
		created.Meta.Name = "cron-sync-test-updated"
		_, err = actions.UpdateWorkflow(ctx, created.ID, created)
		if err != nil {
			t.Fatalf("Failed to update workflow: %v", err)
		}
	})

	t.Run("DisableCronViaUpdate", func(t *testing.T) {
		wf := &models.Workflow{ID: "s1", Meta: models.WorkflowV1Meta{
			Name:             "disable-cron-test",
			Enabled:          true,
			CronEnabled:      true,
			CronExpr:         "@every 2h",
			ServiceAccountID: "sa_trigger",
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock", Params: map[string]string{}},
			},
		}}

		created, err := actions.CreateWorkflow(ctx, wf)
		if err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		// Disable cron
		created.Meta.CronEnabled = false
		updated, err := actions.UpdateWorkflow(ctx, created.ID, created)
		if err != nil {
			t.Fatalf("Failed to update workflow: %v", err)
		}
		if updated.Meta.CronEnabled {
			t.Error("Expected CronEnabled to be false after update")
		}
	})

	t.Run("DeleteWorkflowRemovesTrigger", func(t *testing.T) {
		wf := &models.Workflow{ID: "s1", Meta: models.WorkflowV1Meta{
			Name:             "delete-trigger-test",
			Enabled:          true,
			CronEnabled:      true,
			CronExpr:         "@every 30m",
			ServiceAccountID: "sa_trigger",
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock", Params: map[string]string{}},
			},
		}}

		created, err := actions.CreateWorkflow(ctx, wf)
		if err != nil {
			t.Fatalf("Failed to create workflow: %v", err)
		}

		err = actions.DeleteWorkflow(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to delete workflow: %v", err)
		}

		// Verify workflow is gone
		_, err = actions.GetWorkflow(ctx, created.ID)
		if err == nil {
			t.Error("Expected error when getting deleted workflow")
		}
	})
}

func TestDistributedExecutorConcurrencyCheck(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := tests.SetupMockRootContext()
	_, _ = rbac.CreateServiceAccount(ctx, &models.ServiceAccount{ID: "sa_exec", Meta: models.ServiceAccountV1Meta{Name: "SA Exec"}})

	actions.Register(&tests.MockProcessor{
		ExecuteFunc: func(ctx *actions.TaskContext, inputs map[string]string) (map[string]string, error) {
			// Simulate a long-running step
			time.Sleep(200 * time.Millisecond)
			return map[string]string{}, nil
		},
	})

	t.Run("RejectConcurrentExecution", func(t *testing.T) {
		wf := &models.Workflow{ID: "concurrent-test-wf", Meta: models.WorkflowV1Meta{
			Name:             "Concurrent Test",
			Enabled:          true,
			ServiceAccountID: "sa_exec",
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock", Params: map[string]string{}},
			},
		}}

		// First execution should succeed
		id1, err := actions.GlobalExecutor.Execute(ctx, "root", wf, "Manual", nil, "")
		if err != nil {
			t.Fatalf("First execution failed: %v", err)
		}
		if id1 == "" {
			t.Fatal("Expected non-empty instance ID")
		}

		// Second execution of the same workflow should be rejected (local check)
		_, err = actions.GlobalExecutor.Execute(ctx, "root", wf, "Manual", nil, "")
		if err == nil {
			t.Error("Expected concurrent execution to be rejected")
		}

		// Wait for first to complete
		for i := 0; i < 30; i++ {
			inst, _ := actions.GetTaskInstance(ctx, id1)
			if inst != nil && inst.Status.Status != "Running" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}
	})
}

func TestNotifyClusterNilSubscriber(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// Ensure Subscriber is nil in test environment
	oldSub := common.Subscriber
	common.Subscriber = nil
	defer func() { common.Subscriber = oldSub }()

	// This should not panic even with nil subscriber
	common.NotifyCluster(context.Background(), "test_event", "test_payload")
}

func TestStartEventLoopWithNilSubscriber(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	oldSub := common.Subscriber
	common.Subscriber = nil
	defer func() { common.Subscriber = oldSub }()

	// Should not panic
	common.StartEventLoop(context.Background())
}
