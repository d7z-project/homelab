package actions

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/actions"
)

func GetTaskLogs(ctx context.Context, id string) ([]models.LogEntry, error) {
	instance, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check permission for the parent workflow
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + instance.WorkflowID) {
		return nil, fmt.Errorf("permission denied: actions/%s", instance.WorkflowID)
	}

	// Read all logs from VFS
	logs, err := ReadAllTaskLogs(instance.WorkflowID, id)
	if err != nil {
		return []models.LogEntry{}, nil
	}

	return logs, nil
}

func GetStepLogs(ctx context.Context, id string, stepIndex int, offset int) ([]models.LogEntry, int, error) {
	instance, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return nil, 0, err
	}

	// Check permission for the parent workflow
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + instance.WorkflowID) {
		return nil, 0, fmt.Errorf("permission denied: actions/%s", instance.WorkflowID)
	}

	return ReadStepLogs(instance.WorkflowID, id, stepIndex, offset)
}
