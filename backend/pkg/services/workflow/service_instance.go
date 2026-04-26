package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	repo "homelab/pkg/repositories/workflow/actions"
	"regexp"
	"time"

	"homelab/pkg/models/shared"
	workflowmodel "homelab/pkg/models/workflow"
	runtimepkg "homelab/pkg/runtime"
)

func TriggerWorkflow(ctx context.Context, workflow *workflowmodel.Workflow, userID string, triggerSource string, inputs map[string]string) (string, error) {
	// manual trigger is also blocked if workflow is disabled
	if !workflow.Meta.Enabled {
		return "", fmt.Errorf("workflow is disabled")
	}

	// Permission check for the workflow itself
	if triggerSource == "Manual" {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + workflow.ID) {
			return "", fmt.Errorf("permission denied: actions/%s", workflow.ID)
		}
	}

	// Validate and merge inputs
	if inputs == nil {
		inputs = make(map[string]string)
	}
	finalInputs := make(map[string]string)
	for k, def := range workflow.Meta.Vars {
		val, ok := inputs[k]
		if !ok || val == "" {
			if def.Required && def.Default == "" {
				return "", fmt.Errorf("missing required variable: %s", k)
			}
			val = def.Default
		}
		// Regex validation
		if def.RegexBackend != "" {
			// Skip validation if optional and empty
			if !def.Required && val == "" {
				// OK
			} else {
				matched, err := regexp.MatchString(def.RegexBackend, val)
				if err != nil {
					return "", fmt.Errorf("invalid regex for variable %s: %v", k, err)
				}
				if !matched {
					return "", fmt.Errorf("variable %s does not match required format", k)
				}
			}
		}
		finalInputs[k] = val
	}

	instanceID := fmt.Sprintf("%s%d", TaskPrefix, time.Now().UnixNano())
	instance := &workflowmodel.TaskInstance{
		ID: instanceID,
		Meta: workflowmodel.TaskInstanceV1Meta{
			WorkflowID:       workflow.ID,
			Trigger:          triggerSource,
			UserID:           userID,
			ServiceAccountID: workflow.Meta.ServiceAccountID,
			Inputs:           finalInputs,
			Steps:            make([]workflowmodel.Step, len(workflow.Meta.Steps)),
		},
		Status: workflowmodel.TaskInstanceV1Status{
			Status:      shared.TaskStatusPending,
			StartedAt:   time.Now(),
			Outputs:     make(map[string]string),
			StepTimings: make(map[int]*workflowmodel.StepTiming),
		},
	}
	copy(instance.Meta.Steps, workflow.Meta.Steps)

	if err := repo.SaveTaskInstance(ctx, instance); err != nil {
		return "", fmt.Errorf("failed to save pending instance: %v", err)
	}

	dispatchQueue := runtimepkg.QueueFromContext(ctx)
	if dispatchQueue == nil {
		return "", fmt.Errorf("task queue is not configured")
	}
	payload, err := json.Marshal(workflowmodel.WorkflowExecuteJob{
		WorkflowID: workflow.ID,
		InstanceID: instanceID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to encode workflow dispatch job: %w", err)
	}
	messageID, err := dispatchQueue.Publish(ctx, workflowExecuteTopic, string(payload), nil)
	if err != nil {
		now := time.Now()
		instance.Status.Status = shared.TaskStatusFailed
		instance.Status.Error = fmt.Sprintf("failed to enqueue workflow execution: %v", err)
		instance.Status.FinishedAt = &now
		_ = repo.SaveTaskInstance(ctx, instance)
		return "", fmt.Errorf("failed to enqueue workflow execution: %w", err)
	}
	queuedAt := time.Now()
	instance.Status.QueueTopic = workflowExecuteTopic
	instance.Status.QueueMessageID = messageID
	instance.Status.QueuedAt = &queuedAt
	if err := repo.SaveTaskInstance(ctx, instance); err != nil {
		return "", fmt.Errorf("failed to persist workflow queue metadata: %w", err)
	}

	message := fmt.Sprintf("%s triggered workflow %s (Instance: %s)", triggerSource, workflow.Meta.Name, instanceID)
	commonaudit.FromContext(ctx).Log("TriggerWorkflow", workflow.ID, message, true)
	return instanceID, nil
}

func RunWorkflow(ctx context.Context, workflowID string, inputs map[string]string, triggerSource string) (string, error) {
	// Use distributed lock to prevent concurrent triggers for the same workflow
	lockKey := "action:trigger:" + workflowID
	release := runtimepkg.LockerFromContext(ctx).TryLock(ctx, lockKey)
	if release == nil {
		return "", fmt.Errorf("workflow '%s' is already being triggered, please wait", workflowID)
	}
	defer release()

	workflow, err := repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return "", err
	}

	// Explicit permission check for execution
	perms := commonauth.PermissionsFromContext(ctx)
	if !perms.IsAllowed("actions/" + workflowID) {
		return "", fmt.Errorf("%w: actions/%s (execution access required)", commonauth.ErrPermissionDenied, workflowID)
	}

	authCtx := commonauth.FromContext(ctx)
	userID := "anonymous"
	if authCtx != nil {
		if authCtx.Type == "root" {
			userID = "root"
		} else {
			userID = authCtx.ID
		}
	}

	if triggerSource == "" {
		triggerSource = "Manual"
	}

	return TriggerWorkflow(ctx, workflow, userID, triggerSource, inputs)
}

func GetTaskInstance(ctx context.Context, id string) (*workflowmodel.TaskInstance, error) {
	inst, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	// Check permission for the parent workflow
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + inst.Meta.WorkflowID) {
		return nil, fmt.Errorf("permission denied: actions/%s", inst.Meta.WorkflowID)
	}

	// Populate logs from all parts
	logs, _ := ReadAllTaskLogs(ctx, inst.Meta.WorkflowID, id)
	if logs != nil {
		inst.Status.Logs = logs
	} else {
		inst.Status.Logs = []workflowmodel.LogEntry{}
	}

	return inst, nil
}

func ScanTaskInstances(ctx context.Context, cursor string, limit int, search string, workflowId string) (*shared.PaginationResponse[workflowmodel.TaskInstance], error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions") {
		return nil, fmt.Errorf("%w: actions", commonauth.ErrPermissionDenied)
	}
	res, err := repo.ScanTaskInstances(ctx, cursor, limit, search, workflowId)
	if err != nil {
		return nil, err
	}
	for i := range res.Items {
		// Populate logs from all parts
		logs, _ := ReadAllTaskLogs(ctx, res.Items[i].Meta.WorkflowID, res.Items[i].ID)
		if logs != nil {
			res.Items[i].Status.Logs = logs
		} else {
			res.Items[i].Status.Logs = []workflowmodel.LogEntry{}
		}
	}
	return res, nil
}

func DeleteTaskInstance(ctx context.Context, id string) error {
	inst, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return err
	}

	// Permission check for the parent workflow
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + inst.Meta.WorkflowID) {
		return fmt.Errorf("%w: actions/%s (write access required)", commonauth.ErrPermissionDenied, inst.Meta.WorkflowID)
	}

	// Don't allow deleting running tasks
	if inst.Status.Status == "Running" {
		return fmt.Errorf("cannot delete a running task instance")
	}

	if err := repo.DeleteTaskInstance(ctx, id); err != nil {
		return err
	}

	// Also remove logs
	_ = RemoveTaskLogs(ctx, inst.Meta.WorkflowID, id)
	return nil
}

func CleanupTaskInstances(ctx context.Context, days int) (int, error) {
	all, err := repo.ScanAllTaskInstances(ctx)
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	count := 0
	perms := commonauth.PermissionsFromContext(ctx)

	for _, inst := range all {
		// Only cleanup instances we have permission for
		if !perms.IsAllowed("actions/" + inst.Meta.WorkflowID) {
			continue
		}

		// Only cleanup non-running instances older than cutoff
		if inst.Status.Status != "Running" && inst.Status.StartedAt.Before(cutoff) {
			_ = repo.DeleteTaskInstance(ctx, inst.ID)
			_ = RemoveTaskLogs(ctx, inst.Meta.WorkflowID, inst.ID)
			count++
		}
	}
	return count, nil
}

func CancelTaskInstance(ctx context.Context, id string) error {
	instance, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return err
	}

	// Check permission for the parent workflow
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + instance.Meta.WorkflowID) {
		return fmt.Errorf("%w: actions/%s (write access required)", commonauth.ErrPermissionDenied, instance.Meta.WorkflowID)
	}

	message := fmt.Sprintf("Requested cancellation of task instance %s", id)
	if MustRuntime(ctx).Executor.Cancel(id) {
		commonaudit.FromContext(ctx).Log("CancelTask", id, message, true)
		return nil
	}
	// If not running, maybe it's already finished or doesn't exist
	instance, err = repo.GetTaskInstance(ctx, id)
	if err != nil {
		commonaudit.FromContext(ctx).Log("CancelTask", id, message+" Error: instance not found", false)
		return err
	}
	if instance.Status.Status == "Running" {
		instance.Status.Status = "Cancelled"
		now := time.Now()
		instance.Status.FinishedAt = &now
		if err := repo.SaveTaskInstance(ctx, instance); err != nil {
			commonaudit.FromContext(ctx).Log("CancelTask", id, message+" Error: "+err.Error(), false)
			return err
		}
		commonaudit.FromContext(ctx).Log("CancelTask", id, message, true)
		return nil
	}
	commonaudit.FromContext(ctx).Log("CancelTask", id, message+" (Task not in running state)", true)
	return nil
}
