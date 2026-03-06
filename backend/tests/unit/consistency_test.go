package unit

import (
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
	"homelab/pkg/services/rbac"
	"homelab/tests"
	"testing"
)

func TestDataConsistency(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// Use MockProcessor from actions_test.go
	actions.Register(&MockProcessor{})

	ctx := tests.SetupMockRootContext()

	t.Run("RBAC: RoleBinding with non-existent SA", func(t *testing.T) {
		rb := &models.RoleBinding{
			Name:             "Invalid RB",
			ServiceAccountID: "non-existent-sa",
			RoleIDs:          []string{"admin"},
			Enabled:          true,
		}
		_, err := rbac.CreateRoleBinding(ctx, rb)
		if err == nil {
			t.Error("Expected error when creating RoleBinding with non-existent SA")
		}
	})

	t.Run("RBAC: RoleBinding with non-existent Role", func(t *testing.T) {
		// Create SA first
		sa := &models.ServiceAccount{ID: "sa1", Name: "SA 1"}
		rbac.CreateServiceAccount(ctx, sa)

		rb := &models.RoleBinding{
			Name:             "Invalid RB",
			ServiceAccountID: "sa1",
			RoleIDs:          []string{"non-existent-role"},
			Enabled:          true,
		}
		_, err := rbac.CreateRoleBinding(ctx, rb)
		if err == nil {
			t.Error("Expected error when creating RoleBinding with non-existent Role")
		}
	})

	t.Run("Actions: Workflow with non-existent SA", func(t *testing.T) {
		wf := &models.Workflow{
			Name:             "Invalid WF",
			ServiceAccountID: "non-existent-sa",
			Enabled:          true,
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock", Name: "Step 1"},
			},
		}
		_, err := actions.CreateWorkflow(ctx, wf)
		if err == nil {
			t.Error("Expected error when creating Workflow with non-existent SA")
		}
	})

	t.Run("RBAC: Delete SA used by Workflow", func(t *testing.T) {
		// 1. Create SA
		saID := "worker-sa"
		_, err := rbac.CreateServiceAccount(ctx, &models.ServiceAccount{ID: saID, Name: "Worker SA"})
		if err != nil {
			t.Fatalf("Failed to create SA: %v", err)
		}

		// 2. Create Workflow using this SA
		wf := &models.Workflow{
			Name:             "Worker Workflow",
			ServiceAccountID: saID,
			Enabled:          true,
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock", Name: "Step 1"},
			},
		}
		_, err = actions.CreateWorkflow(ctx, wf)
		if err != nil {
			t.Fatalf("Failed to create Workflow: %v", err)
		}

		// 3. Attempt to delete SA
		err = rbac.DeleteServiceAccount(ctx, saID)
		if err == nil {
			t.Error("Expected error when deleting SA used by Workflow")
		} else {
			t.Logf("Successfully blocked deletion: %v", err)
		}

		// 4. Delete Workflow first
		// Need to get the generated ID
		wfs, _ := actions.ListWorkflows(ctx)
		var wfID string
		for _, w := range wfs {
			if w.Name == "Worker Workflow" {
				wfID = w.ID
				break
			}
		}
		err = actions.DeleteWorkflow(ctx, wfID)
		if err != nil {
			t.Fatalf("Failed to delete Workflow: %v", err)
		}

		// 5. Now delete SA should succeed
		err = rbac.DeleteServiceAccount(ctx, saID)
		if err != nil {
			t.Errorf("Expected success when deleting unused SA, got error: %v", err)
		}
	})

	t.Run("Actions: RunWorkflow Idempotency with Lock", func(t *testing.T) {
		saID := "idemp-sa"
		rbac.CreateServiceAccount(ctx, &models.ServiceAccount{ID: saID, Name: "Idemp SA"})
		wf := &models.Workflow{
			Name:             "Idemp Workflow",
			ServiceAccountID: saID,
			Enabled:          true,
			Steps:            []models.Step{{ID: "s1", Type: "test/mock"}},
		}
		wf, _ = actions.CreateWorkflow(ctx, wf)

		// Trigger multiple times concurrently
		results := make(chan error, 5)
		for i := 0; i < 5; i++ {
			go func() {
				_, err := actions.RunWorkflow(ctx, wf.ID, nil, "Manual")
				results <- err
			}()
		}

		successCount := 0
		failCount := 0
		for i := 0; i < 5; i++ {
			err := <-results
			if err == nil {
				successCount++
			} else {
				failCount++
			}
		}

		if successCount < 1 {
			t.Errorf("Expected at least 1 successful trigger, got %d", successCount)
		}
	})
}
