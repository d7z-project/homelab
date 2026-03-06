package unit

import (
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
	_ "homelab/pkg/services/actions/processors"
	"homelab/pkg/services/rbac"
	"homelab/tests"
	"testing"
	"time"
)

func TestActionsFailAndStatus(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := tests.SetupMockRootContext()
	_, _ = rbac.CreateServiceAccount(ctx, &models.ServiceAccount{ID: "sa", Name: "SA"})
	actions.Register(&MockProcessor{})

	t.Run("Step Fail True - Continue on Error", func(t *testing.T) {
		workflow := &models.Workflow{
			ID: "fail-test-wf", Name: "Fail Test", Enabled: true, ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "core/fail", // This step will fail
					Params: map[string]string{"message": "intentional"},
					Fail:   true, // Allow error
				},
				{
					ID:     "s2",
					Type:   "core/logger",
					If:     `${{ steps.s1.status }} == false`, // Reference status
					Params: map[string]string{"message": "s1 failed as expected"},
				},
			},
		}

		instanceID, err := actions.GlobalExecutor.Execute(ctx, "root", workflow, "Manual", nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Wait for completion
		var instance *models.TaskInstance
		for i := 0; i < 20; i++ {
			instance, _ = actions.GetTaskInstance(ctx, instanceID)
			if instance != nil && instance.Status != "Running" {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if instance == nil || instance.Status != "Success" {
			t.Fatalf("Expected status Success due to fail:true, got %v (Error: %s)", instance.Status, instance.Error)
		}
	})

	t.Run("Step Status Mapping in Params", func(t *testing.T) {
		workflow := &models.Workflow{
			ID: "status-map-wf", Name: "Status Map", Enabled: true, ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "core/logger",
					Params: map[string]string{"message": "hi"},
				},
				{
					ID:     "s2",
					Type:   "test/mock",
					Params: map[string]string{"input_val": "status is ${{ steps.s1.status }}"},
				},
			},
		}

		// We'll capture the input via MockProcessor
		var receivedInput string
		mock := &MockProcessor{
			ExecuteFunc: func(ctx *actions.TaskContext, inputs map[string]string) (map[string]string, error) {
				receivedInput = inputs["input_val"]
				return nil, nil
			},
		}
		actions.Register(mock)

		instanceID, _ := actions.GlobalExecutor.Execute(ctx, "root", workflow, "Manual", nil)

		// Wait for completion
		for i := 0; i < 20; i++ {
			inst, _ := actions.GetTaskInstance(ctx, instanceID)
			if inst != nil && inst.Status != "Running" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		if receivedInput != "status is true" {
			t.Errorf("Expected status to be string 'true' in params, got: %q", receivedInput)
		}
	})
}
