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
		ID:          "test/mock",
		Description: "A mock processor for testing.",
		Params: []models.ParamDefinition{
			{Name: "input_val", Description: "Test input", Optional: false},
		},
		OutputParams: []models.ParamDefinition{
			{Name: "out_val", Description: "Test output"},
		},
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
			ID:               "test-wf",
			Name:             "Test Workflow",
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "step1",
					Type:   "test/mock",
					Name:   "Step 1",
					If:     "",
					Params: map[string]string{"input_val": "hello"},
				},
				{
					ID:     "step2",
					Type:   "test/mock",
					Name:   "Step 2",
					If:     "",
					Params: map[string]string{"input_val": "${{ steps.step1.outputs.out_val }}"},
				},
			},
		}

		instanceID, err := orchestration.GlobalExecutor.Execute(context.Background(), "test-user", workflow, nil)
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

	t.Run("If Condition Evaluation", func(t *testing.T) {
		step1Executed := false
		step2Executed := false
		mock.ExecuteFunc = func(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
			if inputs["input_val"] == "run_me" {
				step1Executed = true
			}
			if inputs["input_val"] == "skip_me" {
				step2Executed = true
			}
			return nil, nil
		}

		workflow := &models.Workflow{
			ID:               "if-wf",
			Name:             "If Workflow",
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "step1",
					Type:   "test/mock",
					If:     "true",
					Params: map[string]string{"input_val": "run_me"},
				},
				{
					ID:     "step2",
					Type:   "test/mock",
					If:     "1 == 2",
					Params: map[string]string{"input_val": "skip_me"},
				},
			},
		}

		instanceID, err := orchestration.GlobalExecutor.Execute(context.Background(), "root", workflow, nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Wait for completion
		for i := 0; i < 10; i++ {
			inst, _ := orchestration.GetTaskInstance(context.Background(), instanceID)
			if inst != nil && inst.Status == "Success" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		if !step1Executed {
			t.Error("Expected step1 to execute")
		}
		if step2Executed {
			t.Error("Expected step2 to be skipped")
		}
	})

	t.Run("Concurrency Control", func(t *testing.T) {
		mock.ExecuteFunc = func(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
			time.Sleep(1 * time.Second)
			return nil, nil
		}

		workflow := &models.Workflow{
			ID:               "concurrent-wf",
			Name:             "Concurrent Workflow",
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock"},
			},
		}

		// Start first instance
		id1, err := orchestration.GlobalExecutor.Execute(context.Background(), "root", workflow, nil)
		if err != nil {
			t.Fatalf("First execution failed: %v", err)
		}
		defer orchestration.GlobalExecutor.Cancel(id1)

		// Try to start second instance immediately
		_, err = orchestration.GlobalExecutor.Execute(context.Background(), "root", workflow, nil)
		if err == nil {
			t.Error("Expected second execution to fail due to concurrency control")
		}
	})

	t.Run("Timeout Mechanism", func(t *testing.T) {
		mock.ExecuteFunc = func(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
			select {
			case <-ctx.Context.Done():
				return nil, ctx.Context.Err()
			case <-time.After(2 * time.Second):
				return nil, nil
			}
		}

		workflow := &models.Workflow{
			ID:               "timeout-wf",
			Name:             "Timeout Workflow",
			ServiceAccountID: "sa",
			Timeout:          1, // 1 second timeout
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock"},
			},
		}

		instanceID, err := orchestration.GlobalExecutor.Execute(context.Background(), "root", workflow, nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Wait for timeout
		var instance *models.TaskInstance
		for i := 0; i < 30; i++ {
			instance, _ = orchestration.GetTaskInstance(context.Background(), instanceID)
			if instance != nil && (instance.Status == "Failed" || instance.Status == "Cancelled") {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if instance == nil || (instance.Status != "Failed" && instance.Status != "Cancelled") {
			t.Errorf("Expected status Failed/Cancelled due to timeout, got %v", instance.Status)
		}
	})

	t.Run("Variable Interpolation", func(t *testing.T) {
		mock.ExecuteFunc = func(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
			return map[string]string{"result": inputs["input_val"] + "-ok"}, nil
		}

		workflow := &models.Workflow{
			ID:               "var-wf",
			Name:             "Var Workflow",
			ServiceAccountID: "sa",
			Vars: map[string]models.VarDefinition{
				"target": {Required: true},
				"opt":    {Required: false},
			},
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "test/mock",
					Name:   "Running for ${{ vars.target }}",
					Params: map[string]string{"input_val": "${{ vars.target }}-${{ vars.opt ?}}-${{ vars.not_exist ?}}-${{ vars.not_exist2 }}"},
				},
			},
		}

		inputs := map[string]string{"target": "PROD", "opt": "yes"}
		instanceID, err := orchestration.TriggerWorkflow(context.Background(), workflow, "root", "Manual", inputs)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Wait for completion
		var instance *models.TaskInstance
		for i := 0; i < 20; i++ {
			instance, _ = orchestration.GetTaskInstance(context.Background(), instanceID)
			if instance != nil && instance.Status == "Success" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		// Verify step name was resolved in logs
		foundLog := false
		for _, l := range instance.Logs {
			if l.Message == "Executing step: Running for PROD (s1)" {
				foundLog = true
			}
		}
		if !foundLog {
			t.Errorf("Step name variable interpolation failed")
		}
	})

	t.Run("Optional Variable Syntax", func(t *testing.T) {
		var receivedInput string
		mock.ExecuteFunc = func(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
			receivedInput = inputs["input_val"]
			return nil, nil
		}

		workflow := &models.Workflow{
			ID:               "opt-wf",
			Name:             "Optional Workflow",
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "test/mock",
					Params: map[string]string{"input_val": "val:${{ vars.missing ?}}-end"},
				},
			},
		}

		instanceID, err := orchestration.TriggerWorkflow(context.Background(), workflow, "root", "Manual", nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Wait for completion
		for i := 0; i < 20; i++ {
			inst, _ := orchestration.GetTaskInstance(context.Background(), instanceID)
			if inst != nil && inst.Status == "Success" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		expected := "val:-end"
		if receivedInput != expected {
			t.Errorf("Expected optional var to resolve to empty, got %q", receivedInput)
		}
	})

	t.Run("Panic Recovery", func(t *testing.T) {
		mock.ExecuteFunc = func(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
			panic("intentional panic for testing")
		}

		workflow := &models.Workflow{
			ID:               "panic-wf",
			Name:             "Panic Workflow",
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock"},
			},
		}

		instanceID, err := orchestration.GlobalExecutor.Execute(context.Background(), "root", workflow, nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Wait for completion
		var instance *models.TaskInstance
		for i := 0; i < 20; i++ {
			instance, _ = orchestration.GetTaskInstance(context.Background(), instanceID)
			if instance != nil && instance.Status == "Failed" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		if instance == nil || instance.Status != "Failed" {
			t.Errorf("Expected status Failed due to panic, got %v", instance.Status)
		}
		if instance.Error == "" {
			t.Error("Expected error message to be recorded")
		}
	})

	t.Run("Status Trigger Control", func(t *testing.T) {
		workflow := &models.Workflow{
			ID:               "status-wf",
			Name:             "Status Workflow",
			ServiceAccountID: "sa",
			Enabled:          false, // Disabled
			Steps:            []models.Step{{ID: "s1", Type: "test/mock"}},
		}

		// 1. TriggerWorkflow (simulating Cron/Webhook) should fail
		_, err := orchestration.TriggerWorkflow(context.Background(), workflow, "cron", "Cron", nil)
		if err == nil {
			t.Error("Expected TriggerWorkflow to fail for disabled workflow")
		}

		// 2. RunWorkflow (Manual) should succeed even if disabled
		ctx := tests.SetupMockRootContext()
		_, err = orchestration.TriggerWorkflow(ctx, workflow, "root", "Manual", nil)
		if err != nil {
			t.Errorf("Expected Manual trigger to work even if disabled, got: %v", err)
		}
	})

	t.Run("ID and Key Validation", func(t *testing.T) {
		ctx := tests.SetupMockRootContext()

		// Invalid Var Key (contains capitals)
		wf1 := &models.Workflow{
			Name:             "Invalid Var",
			ServiceAccountID: "sa",
			Vars: map[string]models.VarDefinition{
				"Invalid_Key": {Required: true},
			},
			Steps: []models.Step{{ID: "s1", Type: "test/mock"}},
		}
		err := orchestration.ValidateWorkflow(ctx, wf1)
		if err == nil {
			t.Error("Expected error for invalid variable key (capitals)")
		}

		// Invalid Step ID (contains capitals)
		wf2 := &models.Workflow{
			Name:             "Invalid Step",
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{ID: "Step_1", Type: "test/mock"},
			},
		}
		err = orchestration.ValidateWorkflow(ctx, wf2)
		if err == nil {
			t.Error("Expected error for invalid step ID (capitals)")
		}

		// Valid
		wf3 := &models.Workflow{
			Name:             "Valid Workflow",
			ServiceAccountID: "sa",
			Vars: map[string]models.VarDefinition{
				"valid_key_123": {Required: true},
			},
			Steps: []models.Step{
				{ID: "valid_step_id", Type: "test/mock"},
			},
		}
		err = orchestration.ValidateWorkflow(ctx, wf3)
		if err != nil {
			t.Errorf("Expected no error for valid IDs, got: %v", err)
		}
	})

	t.Run("RBAC Filtering", func(t *testing.T) {
		// Create 2 workflows (IDs will be generated)
		wf1 := &models.Workflow{Name: "WF 1", ServiceAccountID: "sa", Steps: []models.Step{{ID: "s1", Type: "test/mock"}}}
		wf2 := &models.Workflow{Name: "WF 2", ServiceAccountID: "sa", Steps: []models.Step{{ID: "s1", Type: "test/mock"}}}
		var err error
		wf1, err = orchestration.CreateWorkflow(tests.SetupMockRootContext(), wf1)
		if err != nil {
			t.Fatalf("Failed to create wf1: %v", err)
		}
		wf2, err = orchestration.CreateWorkflow(tests.SetupMockRootContext(), wf2)
		if err != nil {
			t.Fatalf("Failed to create wf2: %v", err)
		}

		// Mock user with permission only for wf1.ID
		userCtx := tests.SetupMockContext("user1", []models.PolicyRule{
			{Resource: "orchestration/" + wf1.ID, Verbs: []string{"get", "list"}},
		})

		// ListWorkflows should only return wf1
		list, _ := orchestration.ListWorkflows(userCtx)
		if len(list) != 1 || list[0].ID != wf1.ID {
			t.Errorf("Expected 1 workflow (wf1), got %d", len(list))
		}

		// GetWorkflow wf2 should fail
		_, err = orchestration.GetWorkflow(userCtx, wf2.ID)
		if err == nil {
			t.Error("Expected GetWorkflow wf2 to fail due to RBAC")
		}
	})

	t.Run("Webhook Token Reset", func(t *testing.T) {
		wf := &models.Workflow{
			ID:               "webhook-wf",
			Name:             "Webhook WF",
			ServiceAccountID: "sa",
			WebhookEnabled:   true,
			Steps:            []models.Step{{ID: "s1", Type: "test/mock"}},
		}
		var err error
		wf, err = orchestration.CreateWorkflow(tests.SetupMockRootContext(), wf)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}

		initialToken := wf.WebhookToken
		if initialToken == "" {
			t.Fatal("Initial token should be generated")
		}

		// Reset token
		newToken, err := orchestration.ResetWebhookToken(tests.SetupMockRootContext(), wf.ID)
		if err != nil {
			t.Fatalf("Reset failed: %v", err)
		}

		if newToken == initialToken {
			t.Error("Token was not changed after reset")
		}

		// Verify in repo
		updated, _ := orchestration.GetWorkflow(tests.SetupMockRootContext(), wf.ID)
		if updated.WebhookToken != newToken {
			t.Error("Token in repo does not match new token")
		}
	})
}
