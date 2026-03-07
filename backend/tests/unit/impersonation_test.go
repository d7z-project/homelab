package unit

import (
	"context"
	"homelab/pkg/common/auth"
	"homelab/pkg/models"
	"homelab/pkg/services/actions"
	"homelab/pkg/services/rbac"
	"homelab/tests"
	"testing"
	"time"
)

type ImpersonationMockProcessor struct {
	CapturedAuth *auth.AuthContext
}

func (m *ImpersonationMockProcessor) Manifest() actions.StepManifest {
	return actions.StepManifest{
		ID:          "test/impersonation_mock",
		Name:        "Impersonation Mock",
		Description: "Captures the auth context for verification.",
	}
}

func (m *ImpersonationMockProcessor) Execute(ctx *actions.TaskContext, inputs map[string]string) (map[string]string, error) {
	m.CapturedAuth = auth.FromContext(ctx.Context)
	return nil, nil
}

func TestImpersonation(t *testing.T) {
	teardown := tests.SetupTestDB()
	defer teardown()

	// 1. Setup Service Account
	saID := "test-executor-sa"
	_, err := rbac.CreateServiceAccount(tests.SetupMockRootContext(), &models.ServiceAccount{
		ID:   saID,
		Name: "Executor SA",
	})
	if err != nil {
		t.Fatalf("Failed to create service account: %v", err)
	}

	// 2. Register Mock Processor
	mock := &ImpersonationMockProcessor{}
	actions.Register(mock)

	// 3. Define Workflow with SA
	workflow := &models.Workflow{
		ID:               "impersonation-wf",
		Name:             "Impersonation Test Workflow",
		Enabled:          true,
		ServiceAccountID: saID,
		Steps: []models.Step{
			{
				ID:   "step1",
				Type: "test/impersonation_mock",
				Name: "Check Identity",
			},
		},
	}

	// 4. Trigger workflow as a different user
	triggerUserID := "human-trigger-user"
	instanceID, err := actions.GlobalExecutor.Execute(context.Background(), triggerUserID, workflow, "Manual", nil, "")
	if err != nil {
		t.Fatalf("Failed to trigger workflow: %v", err)
	}

	// 5. Wait for completion
	var instance *models.TaskInstance
	for i := 0; i < 50; i++ {
		instance, _ = actions.GetTaskInstance(tests.SetupMockRootContext(), instanceID)
		if instance != nil && (instance.Status == "Success" || instance.Status == "Failed") {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if instance == nil || instance.Status != "Success" {
		t.Fatalf("Workflow execution failed or timed out: %v (Error: %s)", instance.Status, instance.Error)
	}

	// 6. Verify Impersonation Result
	if mock.CapturedAuth == nil {
		t.Fatal("Mock processor did not capture auth context")
	}

	t.Logf("Captured Auth ID: %s, Type: %s", mock.CapturedAuth.ID, mock.CapturedAuth.Type)

	if mock.CapturedAuth.ID != saID {
		t.Errorf("Impersonation failed: expected ID %s, got %s", saID, mock.CapturedAuth.ID)
	}

	if mock.CapturedAuth.Type != "sa" {
		t.Errorf("Impersonation failed: expected Type 'sa', got %s", mock.CapturedAuth.Type)
	}

	// Ensure the original triggerer is still recorded in instance but NOT in execution context
	if instance.UserID != triggerUserID {
		t.Errorf("Original triggerer lost: expected %s, got %s", triggerUserID, instance.UserID)
	}
}
