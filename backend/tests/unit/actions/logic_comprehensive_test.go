package actions_test

import (
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
	_ "homelab/pkg/services/actions/processors"
	"homelab/pkg/services/rbac"
	"homelab/tests"
	"testing"
	"time"
)

func TestActionsComprehensiveLogic(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	ctx := tests.SetupMockRootContext()
	_, _ = rbac.CreateServiceAccount(ctx, &models.ServiceAccount{ID: "sa", Name: "SA"})

	// Ensure MockProcessor is registered (it's in actions_test.go)
	actions.Register(&tests.MockProcessor{})

	t.Run("Multiple Variable Interpolation", func(t *testing.T) {
		var receivedInput string
		mock := &tests.MockProcessor{
			ExecuteFunc: func(ctx *actions.TaskContext, inputs map[string]string) (map[string]string, error) {
				receivedInput = inputs["input_val"]
				return map[string]string{"out": "done"}, nil
			},
		}
		actions.Register(mock)

		workflow := &models.Workflow{
			ID: "multi-interp-wf", Name: "Multi Interp", Enabled: true, ServiceAccountID: "sa",
			Vars: map[string]models.VarDefinition{"prefix": {Required: true}},
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock", Params: map[string]string{"input_val": "step1"}},
				{
					ID:   "s2",
					Type: "test/mock",
					Params: map[string]string{
						"input_val": "${{ vars.prefix }}_${{ steps.s1.outputs.out }}_suffix",
					},
				},
			},
		}

		instanceID, _ := actions.GlobalExecutor.Execute(ctx, "root", workflow, "Manual", map[string]string{"prefix": "val"}, "")

		// Wait for completion
		for i := 0; i < 20; i++ {
			inst, _ := actions.GetTaskInstance(ctx, instanceID)
			if inst != nil && inst.Status != "Running" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		expected := "val_done_suffix"
		if receivedInput != expected {
			t.Errorf("Expected interpolated string %q, got %q", expected, receivedInput)
		}
	})

	t.Run("Conditional Branching based on Status", func(t *testing.T) {
		workflow := &models.Workflow{
			ID: "branch-wf", Name: "Branching", Enabled: true, ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "core/fail",
					Params: map[string]string{"message": "err"},
					Fail:   true, // Continue on error
				},
				{
					ID:     "s2",
					Type:   "core/logger",
					If:     `${{ steps.s1.status }} == false`, // Should run
					Params: map[string]string{"message": "s2"},
				},
				{
					ID:     "s3",
					Type:   "core/logger",
					If:     `${{ steps.s1.status }} == true`, // Should NOT run
					Params: map[string]string{"message": "s3"},
				},
			},
		}

		instanceID, _ := actions.GlobalExecutor.Execute(ctx, "root", workflow, "Manual", nil, "")

		for i := 0; i < 20; i++ {
			inst, _ := actions.GetTaskInstance(ctx, instanceID)
			if inst != nil && inst.Status != "Running" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		inst, _ := actions.GetTaskInstance(ctx, instanceID)
		if inst.CurrentStep < 2 {
			t.Errorf("Workflow didn't progress enough, current step: %d", inst.CurrentStep)
		}
	})

	t.Run("Stop Pipeline on Fatal Error", func(t *testing.T) {
		workflow := &models.Workflow{
			ID: "fatal-wf", Name: "Fatal Stop", Enabled: true, ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "core/fail",
					Params: map[string]string{"message": "stop here"},
					Fail:   false, // Default: stop on error
				},
				{
					ID:     "s2",
					Type:   "core/logger",
					Params: map[string]string{"message": "unreachable"},
				},
			},
		}

		instanceID, _ := actions.GlobalExecutor.Execute(ctx, "root", workflow, "Manual", nil, "")

		var instance *models.TaskInstance
		for i := 0; i < 20; i++ {
			instance, _ = actions.GetTaskInstance(ctx, instanceID)
			if instance != nil && instance.Status != "Running" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		if instance.Status != "Failed" {
			t.Errorf("Expected workflow to fail, got %s", instance.Status)
		}
		if instance.CurrentStep != 1 {
			t.Errorf("Expected CurrentStep to be 1, got %d", instance.CurrentStep)
		}
	})

	t.Run("Context Cancellation Aborts Execution", func(t *testing.T) {
		workflow := &models.Workflow{
			ID: "cancel-wf", Name: "Cancellation", Enabled: true, ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "core/sleep",
					Params: map[string]string{"duration": "5s"},
				},
				{
					ID:     "s2",
					Type:   "core/logger",
					Params: map[string]string{"message": "unreachable"},
				},
			},
		}

		instanceID, err := actions.GlobalExecutor.Execute(ctx, "root", workflow, "Manual", nil, "")
		if err != nil {
			t.Fatalf("Failed to execute: %v", err)
		}

		time.Sleep(100 * time.Millisecond)

		success := actions.GlobalExecutor.Cancel(instanceID)
		if !success {
			t.Fatal("Failed to trigger cancellation")
		}

		var instance *models.TaskInstance
		for i := 0; i < 20; i++ {
			instance, _ = actions.GetTaskInstance(ctx, instanceID)
			if instance != nil && (instance.Status == "Cancelled" || instance.Status == "Failed") {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		if instance.Status != "Cancelled" {
			t.Errorf("Expected status Cancelled, got %s", instance.Status)
		}
	})
}
