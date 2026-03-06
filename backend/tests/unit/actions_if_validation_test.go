package unit

import (
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
	_ "homelab/pkg/services/actions/processors"
	"homelab/pkg/services/rbac"
	"homelab/tests"
	"strings"
	"testing"
)

func TestActionsIfValidation(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// Create common service account for tests
	_, _ = rbac.CreateServiceAccount(tests.SetupMockRootContext(), &models.ServiceAccount{
		ID:   "sa",
		Name: "Test SA",
	})

	// Register mock processor needed for complex if validation
	actions.Register(&MockProcessor{})

	ctx := tests.SetupMockRootContext()

	t.Run("Valid If Condition with Variables", func(t *testing.T) {
		workflow := &models.Workflow{
			Name:             "Valid If WF",
			ServiceAccountID: "sa",
			Vars: map[string]models.VarDefinition{
				"is_prod": {Required: true},
			},
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "core/logger",
					If:     `${{ vars.is_prod }} == "true"`,
					Params: map[string]string{"message": "running in prod"},
				},
			},
		}

		err := actions.ValidateWorkflow(ctx, workflow)
		if err != nil {
			t.Errorf("Expected success for valid if condition, got: %v", err)
		}
	})

	t.Run("Invalid If Condition Syntax", func(t *testing.T) {
		workflow := &models.Workflow{
			Name:             "Invalid Syntax WF",
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "core/logger",
					If:     `!!! invalid syntax !!!`,
					Params: map[string]string{"message": "msg"},
				},
			},
		}

		err := actions.ValidateWorkflow(ctx, workflow)
		if err == nil {
			t.Error("Expected error for invalid if syntax, got nil")
		} else if !strings.Contains(err.Error(), "invalid 'if' expression") {
			t.Errorf("Expected 'invalid if expression' error, got: %v", err)
		}
	})

	t.Run("Reference to Future Step", func(t *testing.T) {
		workflow := &models.Workflow{
			Name:             "Future Step Ref WF",
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "core/logger",
					If:     `${{ steps.s2.outputs.status }} == "ok"`, // s2 is defined later
					Params: map[string]string{"message": "s1"},
				},
				{
					ID:     "s2",
					Type:   "core/logger",
					Params: map[string]string{"message": "s2"},
				},
			},
		}

		err := actions.ValidateWorkflow(ctx, workflow)
		if err == nil {
			t.Error("Expected error for future step reference, got nil")
		} else if !strings.Contains(err.Error(), "references unknown or future step") {
			t.Errorf("Expected 'future step' error, got: %v", err)
		}
	})

	t.Run("Reference to Unknown Variable", func(t *testing.T) {
		workflow := &models.Workflow{
			Name:             "Unknown Var WF",
			ServiceAccountID: "sa",
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "core/logger",
					If:     `${{ vars.undefined_var }} == "true"`,
					Params: map[string]string{"message": "msg"},
				},
			},
		}

		err := actions.ValidateWorkflow(ctx, workflow)
		if err == nil {
			t.Error("Expected error for unknown variable reference, got nil")
		} else if !strings.Contains(err.Error(), "references unknown variable") {
			t.Errorf("Expected 'unknown variable' error, got: %v", err)
		}
	})

	t.Run("Complex Valid If Condition", func(t *testing.T) {
		workflow := &models.Workflow{
			Name:             "Complex If WF",
			ServiceAccountID: "sa",
			Vars: map[string]models.VarDefinition{
				"env": {Required: true},
			},
			Steps: []models.Step{
				{
					ID:     "s1",
					Type:   "test/mock", // test/mock defines 'out_val' output
					Params: map[string]string{"input_val": "init"},
				},
				{
					ID:     "s2",
					Type:   "core/logger",
					If:     `${{ vars.env }} == "prod" && ${{ steps.s1.outputs.out_val }} == "success"`,
					Params: map[string]string{"message": "msg"},
				},
			},
		}

		err := actions.ValidateWorkflow(ctx, workflow)
		if err != nil {
			t.Errorf("Expected success for complex valid if condition, got: %v", err)
		}
	})
}
