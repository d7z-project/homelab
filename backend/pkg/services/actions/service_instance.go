package actions

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/actions"
	"regexp"
	"time"
)

func TriggerWorkflow(ctx context.Context, workflow *models.Workflow, userID string, triggerSource string, inputs map[string]string) (string, error) {
	// manual trigger is also blocked if workflow is disabled
	if !workflow.Enabled {
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
	for k, def := range workflow.Vars {
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
	instance := &models.TaskInstance{
		ID:               instanceID,
		WorkflowID:       workflow.ID,
		Status:           "Pending",
		Trigger:          triggerSource,
		UserID:           userID,
		ServiceAccountID: workflow.ServiceAccountID,
		Inputs:           finalInputs,
		StartedAt:        time.Now(),
		Outputs:          make(map[string]string),
		Steps:            make([]models.Step, len(workflow.Steps)),
		StepTimings:      make(map[int]*models.StepTiming),
	}
	copy(instance.Steps, workflow.Steps)

	if err := repo.SaveTaskInstance(ctx, instance); err != nil {
		return "", fmt.Errorf("failed to save pending instance: %v", err)
	}

	if common.Subscriber != nil {
		payload := models.WorkflowExecutePayload{
			WorkflowID: workflow.ID,
			InstanceID: instanceID,
			UserID:     userID,
			Trigger:    triggerSource,
			Inputs:     finalInputs,
		}
		common.NotifyCluster(ctx, common.EventWorkflowExecute, payload)
	} else {
		// Standalone/Test mode fallback: execute locally
		_, _ = GlobalExecutor.Execute(ctx, userID, workflow, triggerSource, finalInputs, instanceID)
	}

	message := fmt.Sprintf("%s triggered workflow %s (Instance: %s)", triggerSource, workflow.Name, instanceID)
	commonaudit.FromContext(ctx).Log("TriggerWorkflow", workflow.ID, message, true)
	return instanceID, nil
}

func RunWorkflow(ctx context.Context, workflowID string, inputs map[string]string, triggerSource string) (string, error) {
	// Use distributed lock to prevent concurrent triggers for the same workflow
	lockKey := "action:trigger:" + workflowID
	release := common.Locker.TryLock(ctx, lockKey)
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

func GetTaskInstance(ctx context.Context, id string) (*models.TaskInstance, error) {
	inst, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	// Check permission for the parent workflow
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + inst.WorkflowID) {
		return nil, fmt.Errorf("permission denied: actions/%s", inst.WorkflowID)
	}

	// Populate logs from all parts
	logs, _ := ReadAllTaskLogs(inst.WorkflowID, id)
	if logs != nil {
		inst.Logs = logs
	} else {
		inst.Logs = []models.LogEntry{}
	}

	return inst, nil
}

func ListTaskInstances(ctx context.Context) ([]models.TaskInstance, error) {
	res, err := repo.ScanTaskInstances(ctx, "", 10000, "")
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	var filtered []models.TaskInstance
	for _, inst := range res.Items {
		if perms.IsAllowed("actions/" + inst.WorkflowID) {
			// Populate logs from all parts
			logs, _ := ReadAllTaskLogs(inst.WorkflowID, inst.ID)
			if logs != nil {
				inst.Logs = logs
			} else {
				inst.Logs = []models.LogEntry{}
			}
			filtered = append(filtered, inst)
		}
	}
	return filtered, nil
}

func ScanTaskInstances(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.TaskInstance], error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions") {
		return nil, fmt.Errorf("%w: actions", commonauth.ErrPermissionDenied)
	}
	res, err := repo.ScanTaskInstances(ctx, cursor, limit, search)
	if err != nil {
		return nil, err
	}
	for i := range res.Items {
		// Populate logs from all parts
		logs, _ := ReadAllTaskLogs(res.Items[i].WorkflowID, res.Items[i].ID)
		if logs != nil {
			res.Items[i].Logs = logs
		} else {
			res.Items[i].Logs = []models.LogEntry{}
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
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + inst.WorkflowID) {
		return fmt.Errorf("%w: actions/%s (write access required)", commonauth.ErrPermissionDenied, inst.WorkflowID)
	}

	// Don't allow deleting running tasks
	if inst.Status == "Running" {
		return fmt.Errorf("cannot delete a running task instance")
	}

	if err := repo.DeleteTaskInstance(ctx, id); err != nil {
		return err
	}

	// Also remove logs
	_ = RemoveTaskLogs(inst.WorkflowID, id)
	return nil
}

func CleanupTaskInstances(ctx context.Context, days int) (int, error) {
	all, err := repo.ListTaskInstances(ctx)
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	count := 0
	perms := commonauth.PermissionsFromContext(ctx)

	for _, inst := range all {
		// Only cleanup instances we have permission for
		if !perms.IsAllowed("actions/" + inst.WorkflowID) {
			continue
		}

		// Only cleanup non-running instances older than cutoff
		if inst.Status != "Running" && inst.StartedAt.Before(cutoff) {
			_ = repo.DeleteTaskInstance(ctx, inst.ID)
			_ = RemoveTaskLogs(inst.WorkflowID, inst.ID)
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
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + instance.WorkflowID) {
		return fmt.Errorf("%w: actions/%s (write access required)", commonauth.ErrPermissionDenied, instance.WorkflowID)
	}

	message := fmt.Sprintf("Requested cancellation of task instance %s", id)
	if GlobalExecutor.Cancel(id) {
		commonaudit.FromContext(ctx).Log("CancelTask", id, message, true)
		return nil
	}
	// If not running, maybe it's already finished or doesn't exist
	instance, err = repo.GetTaskInstance(ctx, id)
	if err != nil {
		commonaudit.FromContext(ctx).Log("CancelTask", id, message+" Error: instance not found", false)
		return err
	}
	if instance.Status == "Running" {
		instance.Status = "Cancelled"
		now := time.Now()
		instance.FinishedAt = &now
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
