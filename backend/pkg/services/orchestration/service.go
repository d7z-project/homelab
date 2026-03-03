package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/orchestration"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
)

func ValidateWorkflow(ctx context.Context, workflow *models.Workflow) error {
	// Call Bind to ensure structural validation and normalization is applied
	if err := workflow.Bind(nil); err != nil {
		return err
	}

	stepIDs := make(map[string]bool)
	for _, step := range workflow.Steps {
		stepIDs[step.ID] = true
	}

	for _, step := range workflow.Steps {
		if _, ok := GetProcessor(step.Type); !ok {
			return fmt.Errorf("step %s: processor not found: %s", step.ID, step.Type)
		}

		// Validate 'if' condition syntax
		if step.If != "" {
			checkIf := paramRegex.ReplaceAllStringFunc(step.If, func(match string) string {
				return "true" // Placeholder for compilation check
			})
			if _, err := expr.Compile(checkIf, expr.AsBool()); err != nil {
				return fmt.Errorf("step %s: invalid 'if' expression: %v", step.ID, err)
			}
		}

		// Validate params for variable references
		for k, v := range step.Params {
			matches := paramRegex.FindAllStringSubmatch(v, -1)
			for _, match := range matches {
				if len(match) < 5 {
					continue
				}
				sID := match[1]
				varKey := match[3]

				if sID != "" {
					if !stepIDs[sID] {
						return fmt.Errorf("step %s: param %s references unknown step %s", step.ID, k, sID)
					}
				} else if varKey != "" {
					if _, ok := workflow.Vars[varKey]; !ok {
						return fmt.Errorf("step %s: param %s references unknown variable %s", step.ID, k, varKey)
					}
				}
			}
		}
	}
	return nil
}

func CreateWorkflow(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	if err := workflow.Bind(nil); err != nil {
		return nil, err
	}
	if err := ValidateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}
	workflow.ID = uuid.New().String()
	workflow.CreatedAt = time.Now()
	workflow.UpdatedAt = time.Now()

	if workflow.WebhookEnabled && workflow.WebhookToken == "" {
		workflow.WebhookToken = GenerateWebhookToken()
	}

	message := fmt.Sprintf("Created workflow %s (Enabled: %v, Timeout: %ds, SA: %s, Cron: %v, Webhook: %v)", workflow.Name, workflow.Enabled, workflow.Timeout, workflow.ServiceAccountID, workflow.CronEnabled, workflow.WebhookEnabled)
	if err := repo.SaveWorkflow(ctx, workflow); err != nil {
		commonaudit.FromContext(ctx).Log("CreateWorkflow", workflow.ID, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("CreateWorkflow", workflow.ID, message, true)
	GlobalTriggerManager.UpdateTriggers(*workflow)
	return workflow, nil
}

func UpdateWorkflow(ctx context.Context, id string, workflow *models.Workflow) (*models.Workflow, error) {
	if err := workflow.Bind(nil); err != nil {
		return nil, err
	}
	old, err := repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}

	// Permission check: orchestration/<workflow-id>
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("orchestration/" + id) {
		return nil, fmt.Errorf("permission denied: orchestration/%s", id)
	}

	if err := ValidateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}

	changes := []string{}
	if old.Enabled != workflow.Enabled {
		changes = append(changes, fmt.Sprintf("enabled: %v -> %v", old.Enabled, workflow.Enabled))
	}
	if old.Timeout != workflow.Timeout {
		changes = append(changes, fmt.Sprintf("timeout: %d -> %d", old.Timeout, workflow.Timeout))
	}
	if old.Name != workflow.Name {
		changes = append(changes, fmt.Sprintf("name: %s -> %s", old.Name, workflow.Name))
	}
	if old.ServiceAccountID != workflow.ServiceAccountID {
		changes = append(changes, fmt.Sprintf("serviceAccountID: %s -> %s", old.ServiceAccountID, workflow.ServiceAccountID))
	}
	if old.CronEnabled != workflow.CronEnabled {
		changes = append(changes, fmt.Sprintf("cronEnabled: %v -> %v", old.CronEnabled, workflow.CronEnabled))
	}
	if old.WebhookEnabled != workflow.WebhookEnabled {
		changes = append(changes, fmt.Sprintf("webhookEnabled: %v -> %v", old.WebhookEnabled, workflow.WebhookEnabled))
	}
	// (Simplified check for vars change just to log it)
	if len(old.Vars) != len(workflow.Vars) {
		changes = append(changes, "vars defined changed")
	}

	if workflow.WebhookEnabled && workflow.WebhookToken == "" {
		workflow.WebhookToken = old.WebhookToken
		if workflow.WebhookToken == "" {
			workflow.WebhookToken = GenerateWebhookToken()
		}
	}
	workflow.CreatedAt = old.CreatedAt
	workflow.UpdatedAt = time.Now()

	message := fmt.Sprintf("Updated workflow %s", workflow.Name)
	if len(changes) > 0 {
		message += ": " + strings.Join(changes, ", ")
	} else {
		message += " (no major changes)"
	}

	if err := repo.SaveWorkflow(ctx, workflow); err != nil {
		commonaudit.FromContext(ctx).Log("UpdateWorkflow", id, message, false)
		return nil, err
	}
	commonaudit.FromContext(ctx).Log("UpdateWorkflow", id, message, true)
	GlobalTriggerManager.UpdateTriggers(*workflow)
	return workflow, nil
}

func ResetWebhookToken(ctx context.Context, id string) (string, error) {
	wf, err := repo.GetWorkflow(ctx, id)
	if err != nil {
		return "", err
	}

	// Permission check: orchestration/<workflow-id>
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("orchestration/" + id) {
		return "", fmt.Errorf("permission denied: orchestration/%s", id)
	}

	wf.WebhookToken = GenerateWebhookToken()
	wf.UpdatedAt = time.Now()

	message := fmt.Sprintf("Reset webhook token for workflow %s", wf.Name)
	if err := repo.SaveWorkflow(ctx, wf); err != nil {
		commonaudit.FromContext(ctx).Log("ResetWebhookToken", id, message, false)
		return "", err
	}
	commonaudit.FromContext(ctx).Log("ResetWebhookToken", id, message, true)
	return wf.WebhookToken, nil
}

func GetWorkflow(ctx context.Context, id string) (*models.Workflow, error) {
	wf, err := repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}
	// Permission check: orchestration/<workflow-id>
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("orchestration/" + id) {
		return nil, fmt.Errorf("permission denied: orchestration/%s", id)
	}
	return wf, nil
}

func DeleteWorkflow(ctx context.Context, id string) error {
	wf, err := repo.GetWorkflow(ctx, id)
	if err != nil {
		return err
	}

	// Permission check: orchestration/<workflow-id>
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("orchestration/" + id) {
		return fmt.Errorf("permission denied: orchestration/%s", id)
	}

	// Cascade delete instances
	instances, err := repo.ListTaskInstances(ctx)
	if err == nil {
		for _, inst := range instances {
			if inst.WorkflowID == id {
				_ = repo.DeleteTaskInstance(ctx, inst.ID)
			}
		}
	}
	GlobalTriggerManager.RemoveTriggers(id)

	snapshot, _ := json.Marshal(wf)
	message := fmt.Sprintf("Deleted workflow %s. Snapshot: %s", wf.Name, string(snapshot))
	if err := repo.DeleteWorkflow(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteWorkflow", id, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteWorkflow", id, message, true)
	return nil
}

func ListWorkflows(ctx context.Context) ([]models.Workflow, error) {
	all, err := repo.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	var filtered []models.Workflow
	for _, wf := range all {
		if perms.IsAllowed("orchestration/" + wf.ID) {
			filtered = append(filtered, wf)
		}
	}
	return filtered, nil
}

// Task Instance Services

func TriggerWorkflow(ctx context.Context, workflow *models.Workflow, userID string, triggerSource string, inputs map[string]string) (string, error) {
	// Permission check for the workflow itself
	if triggerSource == "Manual" {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("orchestration/" + workflow.ID) {
			return "", fmt.Errorf("permission denied: orchestration/%s", workflow.ID)
		}
	}

	// Manual trigger always allowed, Cron/Webhook only if enabled
	if triggerSource != "Manual" && !workflow.Enabled {
		return "", fmt.Errorf("workflow is disabled")
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
		finalInputs[k] = val
	}

	instanceID, err := GlobalExecutor.Execute(ctx, userID, workflow, finalInputs)
	message := fmt.Sprintf("%s triggered workflow %s (Instance: %s)", triggerSource, workflow.Name, instanceID)
	if err != nil {
		commonaudit.FromContext(ctx).Log("TriggerWorkflow", workflow.ID, message+" Error: "+err.Error(), false)
		return "", err
	}
	commonaudit.FromContext(ctx).Log("TriggerWorkflow", workflow.ID, message, true)
	return instanceID, nil
}

func RunWorkflow(ctx context.Context, workflowID string, inputs map[string]string) (string, error) {
	workflow, err := repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return "", err
	}

	// Explicit permission check
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("orchestration/" + workflowID) {
		return "", fmt.Errorf("permission denied: orchestration/%s", workflowID)
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

	return TriggerWorkflow(ctx, workflow, userID, "Manual", inputs)
}

func GetTaskInstance(ctx context.Context, id string) (*models.TaskInstance, error) {
	inst, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return nil, err
	}
	// Check permission for the parent workflow
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("orchestration/" + inst.WorkflowID) {
		return nil, fmt.Errorf("permission denied: orchestration/%s", inst.WorkflowID)
	}
	return inst, nil
}

func ListTaskInstances(ctx context.Context) ([]models.TaskInstance, error) {
	all, err := repo.ListTaskInstances(ctx)
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	var filtered []models.TaskInstance
	for _, inst := range all {
		if perms.IsAllowed("orchestration/" + inst.WorkflowID) {
			filtered = append(filtered, inst)
		}
	}
	return filtered, nil
}

func CancelTaskInstance(ctx context.Context, id string) error {
	instance, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return err
	}

	// Check permission for the parent workflow
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("orchestration/" + instance.WorkflowID) {
		return fmt.Errorf("permission denied: orchestration/%s", instance.WorkflowID)
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

func GetTaskLogs(ctx context.Context, id string) (string, error) {
	instance, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return "", err
	}

	// Check permission for the parent workflow
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("orchestration/" + instance.WorkflowID) {
		return "", fmt.Errorf("permission denied: orchestration/%s", instance.WorkflowID)
	}

	// Return logs as JSON string
	data, err := json.Marshal(instance.Logs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type ProbeRequest struct {
	ProcessorID string            `json:"processorId"`
	Params      map[string]string `json:"params"`
}

func (p *ProbeRequest) Bind(r *http.Request) error {
	if p.ProcessorID == "" {
		return fmt.Errorf("processorId is required")
	}
	return nil
}

func Probe(ctx context.Context, req *ProbeRequest) (map[string]string, error) {
	processor, ok := GetProcessor(req.ProcessorID)
	if !ok {
		return nil, fmt.Errorf("processor not found: %s", req.ProcessorID)
	}

	// Create a temporary workspace for probe
	workspace, err := os.MkdirTemp("", "probe_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workspace)

	logger := NewTaskLogger()
	defer logger.Close()

	authCtx := commonauth.FromContext(ctx)
	userID := "anonymous"
	if authCtx != nil {
		if authCtx.Type == "root" {
			userID = "root"
		} else {
			userID = authCtx.ID
		}
	}

	taskCtx := &TaskContext{
		InstanceID: "probe",
		Workspace:  workspace,
		UserID:     userID,
		Context:    ctx,
		CancelFunc: func() {},
		Logger:     logger,
	}

	return processor.Execute(taskCtx, req.Params)
}
