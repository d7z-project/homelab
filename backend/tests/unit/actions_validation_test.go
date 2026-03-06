package unit

import (
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
	_ "homelab/pkg/services/actions/processors"
	"homelab/pkg/services/rbac"
	"homelab/tests"
	"strings"
	"testing"
	"time"
)

func TestActionsRegexValidation(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// Create common service account for tests
	_, _ = rbac.CreateServiceAccount(tests.SetupMockRootContext(), &models.ServiceAccount{
		ID:   "sa",
		Name: "Test SA",
	})

	t.Run("Variable Regex Validation", func(t *testing.T) {
		workflow := &models.Workflow{
			ID:               "regex-var-wf",
			Name:             "Regex Var Workflow",
			Enabled:          true,
			ServiceAccountID: "sa",
			Vars: map[string]models.VarDefinition{
				"env": {
					Required:     true,
					RegexBackend: "^(prod|staging)$",
				},
			},
			Steps: []models.Step{
				{ID: "s1", Type: "core/logger", Params: map[string]string{"message": "env is ${{ vars.env }}"}},
			},
		}

		ctx := tests.SetupMockRootContext()

		// Valid input
		_, err := actions.TriggerWorkflow(ctx, workflow, "root", "Manual", map[string]string{"env": "prod"})
		if err != nil {
			t.Errorf("Expected success for valid regex variable, got: %v", err)
		}

		// Invalid input
		_, err = actions.TriggerWorkflow(ctx, workflow, "root", "Manual", map[string]string{"env": "dev"})
		if err == nil {
			t.Error("Expected error for invalid regex variable, got nil")
		}
	})

	t.Run("Step Parameter Regex Validation", func(t *testing.T) {
		workflow := &models.Workflow{
			ID:               "regex-param-wf",
			Name:             "Regex Param Workflow",
			Enabled:          true,
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:   "s1",
					Type: "core/sleep",
					Params: map[string]string{
						"duration": "invalid", // core/sleep has ^\d+[smh]$ regex
					},
				},
			},
		}

		ctx := tests.SetupMockRootContext()
		instanceID, err := actions.GlobalExecutor.Execute(ctx, "root", workflow, "Manual", nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Wait for completion (should fail)
		var instance *models.TaskInstance
		for i := 0; i < 10; i++ {
			instance, _ = actions.GetTaskInstance(ctx, instanceID)
			if instance != nil && instance.Status != "Running" {
				break
			}
			time.Sleep(50 * time.Millisecond)
		}

		if instance == nil || instance.Status != "Failed" {
			t.Errorf("Expected status Failed due to regex mismatch, got %v", instance.Status)
		}
		if instance.Error == "" || !strings.Contains(instance.Error, "does not match required format") {
			t.Errorf("Expected regex validation error message, got: %q", instance.Error)
		}
	})

	t.Run("Failure Step Index Persistence", func(t *testing.T) {
		workflow := &models.Workflow{
			ID:               "fail-step-wf",
			Name:             "Fail Step Workflow",
			Enabled:          true,
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "core/logger",
					Params: map[string]string{"message": "step 1"},
				},
				{
					ID:     "s2",
					Type:   "core/fail",
					Params: map[string]string{"message": "intentional failure"},
				},
				{
					ID:     "s3",
					Type:   "core/logger",
					Params: map[string]string{"message": "step 3"},
				},
			},
		}

		ctx := tests.SetupMockRootContext()
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
			time.Sleep(50 * time.Millisecond)
		}

		if instance == nil || instance.Status != "Failed" {
			t.Fatalf("Expected status Failed, got %v", instance.Status)
		}

		// CurrentStep should be 2 (s2 failed), not 4 (len(Steps) + 1)
		if instance.CurrentStep != 2 {
			t.Errorf("Expected CurrentStep to be 2 (failed step index), got %d", instance.CurrentStep)
		}
	})

	t.Run("ValidateWorkflow Parameter Regex", func(t *testing.T) {
		workflow := &models.Workflow{
			Name:             "Invalid Param WF",
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:   "s1",
					Type: "core/sleep",
					Params: map[string]string{
						"duration": "invalid",
					},
				},
			},
		}

		ctx := tests.SetupMockRootContext()
		err := actions.ValidateWorkflow(ctx, workflow)
		if err == nil {
			t.Error("Expected ValidateWorkflow to fail for invalid static parameter, but it succeeded")
		} else if !strings.Contains(err.Error(), "does not match required format") {
			t.Errorf("Expected regex validation error message, got: %v", err)
		}

		// Template variable should bypass static validation
		workflow.Vars = map[string]models.VarDefinition{
			"timeout": {Required: true},
		}
		workflow.Steps[0].Params["duration"] = "${{ vars.timeout }}"
		err = actions.ValidateWorkflow(ctx, workflow)
		if err != nil {
			t.Errorf("Expected ValidateWorkflow to skip validation for template variable, but it failed: %v", err)
		}
	})

	t.Run("Optional Parameter Regex Bypassing when Empty", func(t *testing.T) {
		teardown := tests.SetupTestDB()
		defer teardown()

		_, _ = rbac.CreateServiceAccount(tests.SetupMockRootContext(), &models.ServiceAccount{
			ID:   "sa",
			Name: "Test SA",
		})

		workflow := &models.Workflow{
			ID:               "optional-regex-wf",
			Name:             "Optional Regex Workflow",
			Enabled:          true,
			ServiceAccountID: "sa",
			Vars: map[string]models.VarDefinition{
				"optional_var": {
					Required:     false,
					RegexBackend: "^[0-9]+$", // Must be digits if provided
				},
			},
			Steps: []models.Step{
				{
					ID:   "s1",
					Type: "core/workflow_call",
					Params: map[string]string{
						"workflow_id": "some-other-id",
						"vars":        "", // Optional JSON, regex ^\{.*\}$
					},
				},
			},
		}

		ctx := tests.SetupMockRootContext()

		// 1. Test ValidateWorkflow (Step Parameters)
		err := actions.ValidateWorkflow(ctx, workflow)
		if err != nil {
			t.Errorf("ValidateWorkflow failed for empty optional parameter: %v", err)
		}

		// 2. Test TriggerWorkflow (Workflow Variables)
		// Empty input for optional_var should succeed despite regex
		_, err = actions.TriggerWorkflow(ctx, workflow, "root", "Manual", map[string]string{"optional_var": ""})
		if err != nil && !strings.Contains(err.Error(), "not found") { // Ignore "target workflow not found" error from the processor execution part
			if strings.Contains(err.Error(), "match required format") {
				t.Errorf("TriggerWorkflow failed regex for empty optional variable: %v", err)
			}
		}
	})
}
