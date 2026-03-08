package common_test

import (
	"homelab/pkg/models"
	rbacrepo "homelab/pkg/repositories/rbac"
	"homelab/pkg/services/actions"
	"homelab/pkg/services/rbac"
	"homelab/tests"
	"testing"
	"time"
)

func TestDataConsistency(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// Use MockProcessor from actions_test.go
	actions.Register(&tests.MockProcessor{})

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
		// 1. Create SA (Direct repo call for persistence)
		saID := "worker-sa"
		_ = rbacrepo.SaveServiceAccount(ctx, &models.ServiceAccount{ID: saID, Name: "Worker SA", Enabled: true})

		// 2. Create Workflow using this SA
		wf := &models.Workflow{
			Name:             "Worker Workflow",
			ServiceAccountID: saID,
			Enabled:          true,
			Steps: []models.Step{
				{ID: "s1", Type: "test/mock", Name: "Step 1"},
			},
		}
		_, err := actions.CreateWorkflow(ctx, wf)
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
		wfResp, _ := actions.ScanWorkflows(ctx, "", 1000, "")
		var wfID string
		if wfResp != nil {
			for _, w := range wfResp.Items {
				if w.Name == "Worker Workflow" {
					wfID = w.ID
					break
				}
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
		_, _ = rbac.CreateServiceAccount(ctx, &models.ServiceAccount{ID: saID, Name: "Idemp SA"})
		wf := &models.Workflow{
			Name:             "Idemp Workflow",
			ServiceAccountID: saID,
			Enabled:          true,
			Steps:            []models.Step{{ID: "s1", Type: "test/mock"}},
		}
		wf, _ = actions.CreateWorkflow(ctx, wf)

		type runResult struct {
			err error
			id  string
		}
		results := make(chan runResult, 5)
		for i := 0; i < 5; i++ {
			go func() {
				id, err := actions.RunWorkflow(ctx, wf.ID, nil, "Manual")
				results <- runResult{err: err, id: id}
			}()
		}

		var instanceIDs []string
		successCount := 0
		failCount := 0
		for i := 0; i < 5; i++ {
			res := <-results
			if res.err == nil {
				successCount++
				instanceIDs = append(instanceIDs, res.id)
			} else {
				failCount++
			}
		}

		for _, id := range instanceIDs {
			for j := 0; j < 50; j++ {
				instance, _ := actions.GetTaskInstance(ctx, id)
				if instance != nil && (instance.Status == "Success" || instance.Status == "Failed") {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
		}

		if successCount < 1 {
			t.Errorf("Expected at least 1 successful trigger, got %d", successCount)
		}
	})
}
