package unit

import (
	"homelab/pkg/models"
	"homelab/pkg/services/orchestration"
	_ "homelab/pkg/services/orchestration/processors"
	"homelab/pkg/services/rbac"
	"homelab/tests"
	"strings"
	"testing"
	"time"
)

func TestOrchestrationRegexValidation(t *testing.T) {
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
		_, err := orchestration.TriggerWorkflow(ctx, workflow, "root", "Manual", map[string]string{"env": "prod"})
		if err != nil {
			t.Errorf("Expected success for valid regex variable, got: %v", err)
		}

		// Invalid input
		_, err = orchestration.TriggerWorkflow(ctx, workflow, "root", "Manual", map[string]string{"env": "dev"})
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
		instanceID, err := orchestration.GlobalExecutor.Execute(ctx, "root", workflow, "Manual", nil)
		if err != nil {
			t.Fatalf("Execute failed: %v", err)
		}

		// Wait for completion (should fail)
		var instance *models.TaskInstance
		for i := 0; i < 10; i++ {
			instance, _ = orchestration.GetTaskInstance(ctx, instanceID)
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
}
