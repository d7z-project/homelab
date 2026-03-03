package unit

import (
	"context"
	"homelab/pkg/models"
	"homelab/pkg/services/orchestration"
	_ "homelab/pkg/services/orchestration/processors"
	"homelab/tests"
	"testing"
	"time"
)

type MockProcessor struct {
	ExecuteFunc func(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error)
}

func (m *MockProcessor) Manifest() orchestration.StepManifest {
	return orchestration.StepManifest{
		ID: "test/mock",
	}
}

func (m *MockProcessor) Execute(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
	return m.ExecuteFunc(ctx, inputs)
}

func TestOrchestrationEngine(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// Register mock
	mock := &MockProcessor{}
	orchestration.Register(mock)

	t.Run("Basic Execution and Parameter Mapping", func(t *testing.T) {
		mock.ExecuteFunc = func(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
			if ctx.InstanceID == "probe" {
				return nil, nil
			}
			val := inputs["input_val"]
			return map[string]string{"out_val": val + "_processed"}, nil
		}

		workflow := &models.Workflow{
			ID:   "test-wf",
			Name: "Test Workflow",
			Steps: []models.Step{
				{
					ID:     "step1",
					Type:   "test/mock",
					Name:   "Step 1",
					Params: map[string]string{"input_val": "hello"},
				},
				{
					ID:     "step2",
					Type:   "test/mock",
					Name:   "Step 2",
					Params: map[string]string{"input_val": "${{ steps.step1.outputs.out_val }}"},
				},
			},
		}

		instanceID, err := orchestration.GlobalExecutor.Execute(context.Background(), "test-user", workflow)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Wait for completion
		var instance *models.TaskInstance
		for i := 0; i < 20; i++ {
			instance, _ = orchestration.GetTaskInstance(context.Background(), instanceID)
			if instance != nil && instance.Status != "Running" {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if instance == nil {
			t.Fatal("Instance not found")
		}
		if instance.Status != "Success" {
			t.Errorf("Expected status Success, got %s (Error: %s)", instance.Status, instance.Error)
		}
	})

	t.Run("Cancellation", func(t *testing.T) {
		mock.ExecuteFunc = func(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
			select {
			case <-ctx.Context.Done():
				return nil, ctx.Context.Err()
			case <-time.After(5 * time.Second):
				return nil, nil
			}
		}

		workflow := &models.Workflow{
			ID:   "cancel-wf",
			Name: "Cancel Workflow",
			Steps: []models.Step{
				{
					ID:   "long-step",
					Type: "test/mock",
					Name: "Long Step",
				},
			},
		}

		instanceID, err := orchestration.GlobalExecutor.Execute(context.Background(), "test-user", workflow)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		time.Sleep(200 * time.Millisecond)
		cancelled := orchestration.GlobalExecutor.Cancel(instanceID)
		if !cancelled {
			t.Fatal("Cancel failed to signal")
		}

		// Wait for status update
		var instance *models.TaskInstance
		for i := 0; i < 20; i++ {
			instance, _ = orchestration.GetTaskInstance(context.Background(), instanceID)
			if instance != nil && (instance.Status == "Cancelled" || instance.Status == "Failed") {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if instance == nil {
			t.Fatal("Instance not found")
		}
		if instance.Status != "Cancelled" && instance.Status != "Failed" {
			t.Errorf("Expected status Cancelled or Failed, got %s", instance.Status)
		}
	})
}
