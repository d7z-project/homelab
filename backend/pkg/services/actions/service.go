package actions

import (
	"context"
	"encoding/json"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/actions"
	"homelab/pkg/services/discovery"
	"homelab/pkg/services/rbac"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
	"github.com/spf13/afero"
)

func ValidateWorkflow(ctx context.Context, workflow *models.Workflow) error {
	// Call Bind to ensure structural validation and normalization is applied
	if err := workflow.Bind(nil); err != nil {
		return err
	}

	// Verify ServiceAccount exists using discovery service
	exists, err := discovery.Verify(ctx, "rbac/serviceaccounts", workflow.ServiceAccountID)
	if err != nil {
		return fmt.Errorf("failed to verify service account: %v", err)
	}
	if !exists {
		return fmt.Errorf("service account '%s' not found", workflow.ServiceAccountID)
	}

	stepIDs := make(map[string]bool)
	// Map to store output parameters for each step for cross-reference validation
	// stepID -> map[paramName]bool
	stepOutputsMap := make(map[string]map[string]bool)

	// Validate variables
	for k, v := range workflow.Vars {
		if !models.ActionIdRegex.MatchString(k) {
			return fmt.Errorf("invalid variable key: %s (must match %s)", k, models.ActionIdRegex.String())
		}
		if v.RegexBackend != "" {
			if _, err := regexp.Compile(v.RegexBackend); err != nil {
				return fmt.Errorf("invalid regex for variable %s: %v", k, err)
			}
		}
	}

	for _, step := range workflow.Steps {
		processor, ok := GetProcessor(step.Type)
		if !ok {
			return fmt.Errorf("step %s: processor not found: %s", step.ID, step.Type)
		}

		manifest := processor.Manifest()

		// 0. Recursion Check (Direct)
		if step.Type == "core/workflow_call" && workflow.ID != "" {
			calledID := step.Params["workflow_id"]
			if calledID == workflow.ID {
				return fmt.Errorf("step %s: recursive workflow call detected (cannot call itself)", step.ID)
			}
		}

		manifestParams := make(map[string]models.ParamDefinition)
		for _, p := range manifest.Params {
			manifestParams[p.Name] = p
		}

		// Record outputs for future steps to reference
		stepOutputsMap[step.ID] = make(map[string]bool)
		for _, op := range manifest.OutputParams {
			stepOutputsMap[step.ID][op.Name] = true
		}

		// 1. Check for required parameters and existence
		for _, pDef := range manifest.Params {
			val, ok := step.Params[pDef.Name]
			if !pDef.Optional {
				if !ok || strings.TrimSpace(val) == "" {
					return fmt.Errorf("step %s: missing required parameter '%s'", step.ID, pDef.Name)
				}
			}
		}

		// 2. Check for undefined parameters
		for k := range step.Params {
			if _, ok := manifestParams[k]; !ok {
				return fmt.Errorf("step %s: undefined parameter '%s'", step.ID, k)
			}
		}

		// 3. Validate 'if' condition syntax and references
		if step.If != "" {
			// Extract all references like ${{ vars.x }} or ${{ steps.id.outputs.y }}
			matches := paramRegex.FindAllStringSubmatch(step.If, -1)

			env := make(map[string]interface{})
			exprStr := step.If

			for i, match := range matches {
				if len(match) < 6 {
					continue
				}
				fullMatch := match[0]
				sID := match[1]
				refType := match[2] // "outputs.KEY" or "status"
				outputKey := match[3]
				varKey := match[4]
				isOptional := match[5] == "?"

				// Check timing and existence
				if sID != "" {
					// Step ID must always exist (temporal check)
					if !stepIDs[sID] {
						return fmt.Errorf("step %s: 'if' condition references unknown or future step '%s'", step.ID, sID)
					}
					// Only strictly check output key if NOT optional AND not "status"
					if !isOptional && refType != "status" && outputKey != "" && !stepOutputsMap[sID][outputKey] {
						return fmt.Errorf("step %s: 'if' condition references unknown output '%s' from step '%s' (use '?' for optional)", step.ID, outputKey, sID)
					}
				} else if varKey != "" {
					// Only strictly check variable existence if NOT optional
					if !isOptional {
						if _, ok := workflow.Vars[varKey]; !ok {
							return fmt.Errorf("step %s: 'if' condition references unknown variable '%s' (use '?' for optional)", step.ID, varKey)
						}
					}
				}

				placeholder := fmt.Sprintf("__v%d", i)
				exprStr = strings.Replace(exprStr, fullMatch, placeholder, 1)
				env[placeholder] = ""
			}
			program, err := expr.Compile(exprStr, expr.Env(env), expr.AsBool())
			if err != nil {
				return fmt.Errorf("step %s: invalid 'if' expression: %v", step.ID, err)
			}
			_ = program
		}

		// Update stepIDs for next steps to reference this one
		stepIDs[step.ID] = true

		// 4. Validate params for variable references and regex
		for k, v := range step.Params {
			pDef := manifestParams[k]

			// Check for variable references
			matches := paramRegex.FindAllStringSubmatch(v, -1)
			for _, match := range matches {
				if len(match) < 6 {
					continue
				}
				sID := match[1]
				refType := match[2] // "outputs.KEY" or "status"
				outputKey := match[3]
				varKey := match[4]
				isOptional := match[5] == "?"

				if sID != "" {
					if !stepIDs[sID] {
						return fmt.Errorf("step %s: param %s references unknown or future step '%s'", step.ID, k, sID)
					}
					// Only strictly check output key if NOT optional AND not "status"
					if !isOptional && refType != "status" && outputKey != "" && !stepOutputsMap[sID][outputKey] {
						return fmt.Errorf("step %s: param %s references unknown output '%s' from step '%s' (use '?' for optional)", step.ID, k, outputKey, sID)
					}
				} else if varKey != "" {
					// Only strictly check variable existence if NOT optional
					if !isOptional {
						if _, ok := workflow.Vars[varKey]; !ok {
							return fmt.Errorf("step %s: param %s references unknown variable %s (use '?' for optional)", step.ID, k, varKey)
						}
					}
				}
			}

			// Regex validation for static values (not containing templates)
			if !strings.Contains(v, "${{") {
				// Lookup validation
				if pDef.LookupCode != "" && v != "" {
					exists, err := discovery.Verify(ctx, pDef.LookupCode, v)
					if err != nil {
						return fmt.Errorf("step %s: failed to verify parameter %s via discovery code %s: %v", step.ID, k, pDef.LookupCode, err)
					}
					if !exists {
						return fmt.Errorf("step %s: parameter %s value '%s' not found in discovery code %s", step.ID, k, v, pDef.LookupCode)
					}
				}

				if pDef.RegexBackend != "" {
					// Skip validation if optional and empty (already handled by required check if not optional)
					if pDef.Optional && v == "" {
						continue
					}
					matched, err := regexp.MatchString(pDef.RegexBackend, v)
					if err != nil {
						return fmt.Errorf("step %s: invalid regex for param %s: %v", step.ID, k, err)
					}
					if !matched {
						return fmt.Errorf("step %s: parameter %s does not match required format", step.ID, k)
					}
				}
			}
		}
	}
	return nil
}

func CreateWorkflow(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
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
	common.NotifyCluster(ctx, "workflow_trigger_update", workflow.ID)
	return workflow, nil
}

func UpdateWorkflow(ctx context.Context, id string, workflow *models.Workflow) (*models.Workflow, error) {
	// 1. Basic permission and existence check
	old, err := repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}

	// Permission check: actions/<workflow-id>
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + id) {
		return nil, fmt.Errorf("%w: actions/%s (write access required)", commonauth.ErrPermissionDenied, id)
	}

	// 2. Ensure ID consistency before validation
	workflow.ID = id

	// 3. Perform unified validation
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
	if old.CronExpr != workflow.CronExpr {
		changes = append(changes, fmt.Sprintf("cronExpr: %s -> %s", old.CronExpr, workflow.CronExpr))
	}
	if old.Description != workflow.Description {
		changes = append(changes, "description changed")
	}
	if old.WebhookEnabled != workflow.WebhookEnabled {
		changes = append(changes, fmt.Sprintf("webhookEnabled: %v -> %v", old.WebhookEnabled, workflow.WebhookEnabled))
	}
	// (Simplified check for vars and steps change)
	if len(old.Vars) != len(workflow.Vars) {
		changes = append(changes, "vars defined changed")
	}
	if len(old.Steps) != len(workflow.Steps) {
		changes = append(changes, "steps changed")
	}

	if workflow.WebhookEnabled && workflow.WebhookToken == "" {
		workflow.WebhookToken = old.WebhookToken
		if workflow.WebhookToken == "" {
			workflow.WebhookToken = GenerateWebhookToken()
		}
	}
	workflow.ID = id
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
	common.NotifyCluster(ctx, "workflow_trigger_update", workflow.ID)
	return workflow, nil
}

func ResetWebhookToken(ctx context.Context, id string) (string, error) {
	wf, err := repo.GetWorkflow(ctx, id)
	if err != nil {
		return "", err
	}

	// Permission check: actions/<workflow-id>
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + id) {
		return "", fmt.Errorf("%w: actions/%s (write access required)", commonauth.ErrPermissionDenied, id)
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
	// Permission check: actions/<workflow-id>
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + id) {
		return nil, fmt.Errorf("permission denied: actions/%s", id)
	}
	return wf, nil
}

func DeleteWorkflow(ctx context.Context, id string) error {
	wf, err := repo.GetWorkflow(ctx, id)
	if err != nil {
		return err
	}

	// Permission check: actions/<workflow-id>
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + id) {
		return fmt.Errorf("%w: actions/%s (write access required)", commonauth.ErrPermissionDenied, id)
	}

	// Cascade delete instances and logs
	instances, err := repo.ListTaskInstances(ctx)
	if err == nil {
		for _, inst := range instances {
			if inst.WorkflowID == id {
				_ = repo.DeleteTaskInstance(ctx, inst.ID)
			}
		}
		_ = RemoveWorkflowLogs(id)
	}
	GlobalTriggerManager.RemoveTriggers(id)

	snapshot, _ := json.Marshal(wf)
	message := fmt.Sprintf("Deleted workflow %s. Snapshot: %s", wf.Name, string(snapshot))
	if err := repo.DeleteWorkflow(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteWorkflow", id, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteWorkflow", id, message, true)
	common.NotifyCluster(ctx, "workflow_trigger_delete", id)
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
		if perms.IsAllowed("actions/" + wf.ID) {
			filtered = append(filtered, wf)
		}
	}
	return filtered, nil
}

// Task Instance Services

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
		b, _ := json.Marshal(payload)
		common.NotifyCluster(ctx, "workflow_execute", string(b))
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
	all, err := repo.ListTaskInstances(ctx)
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	var filtered []models.TaskInstance
	for _, inst := range all {
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

func ValidateRegex(regex string) error {
	if regex == "" {
		return nil
	}
	_, err := regexp.Compile(regex)
	return err
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

	// Generate a unique ID for this probe to avoid log collision
	instanceID := fmt.Sprintf("probe_%d", time.Now().UnixNano())

	// Create a temporary workspace for probe in actionsFS
	workspace, err := afero.TempDir(actionsFS, "", instanceID)
	if err != nil {
		return nil, err
	}
	defer actionsFS.RemoveAll(workspace)

	// Use '_probe' as a reserved workflow ID for system-level tests
	logger, err := NewTaskLogger("_probe", instanceID)
	if err != nil {
		return nil, err
	}
	defer logger.Close()
	defer RemoveTaskLogs("_probe", instanceID)

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
		WorkflowID:       "_probe",
		InstanceID:       instanceID,
		Workspace:        afero.NewBasePathFs(actionsFS, workspace),
		UserID:           userID,
		ServiceAccountID: "root", // Probes run as root for full validation
		Context:          ctx,
		CancelFunc:       func() {},
		Logger:           logger,
	}

	return processor.Execute(taskCtx, req.Params)
}
func init() {
	rbac.RegisterResourceWithVerbs("actions", func(ctx context.Context, prefix string) ([]models.DiscoverResult, error) {

		// prefix is everything after "actions/"
		subs := []string{"workflows", "instances", "manifests", "probe"}
		res := make([]models.DiscoverResult, 0)
		for _, s := range subs {
			if strings.HasPrefix(s, prefix) {
				res = append(res, models.DiscoverResult{
					FullID: s,
					Name:   s,
					Final:  true,
				})
			}
		}

		// If prefix starts with a sub-resource, suggest IDs
		for _, s := range []string{"workflows", "instances"} {
			if strings.HasPrefix(prefix, s+"/") {
				idPrefix := strings.TrimPrefix(prefix, s+"/")
				if s == "workflows" {
					workflows, err := repo.ListWorkflows(ctx)
					if err == nil {
						for _, wf := range workflows {
							if strings.HasPrefix(wf.ID, idPrefix) {
								res = append(res, models.DiscoverResult{
									FullID: "workflows/" + wf.ID,
									Name:   "Workflow: " + wf.Name,
									Final:  true,
								})
							}
						}
					}
				} else {
					res = append(res, models.DiscoverResult{
						FullID: "instances/*",
						Name:   "All Instances",
						Final:  true,
					})
				}
			}
		}

		return res, nil
	}, []string{"get", "list", "create", "update", "delete", "execute", "*"})

	discovery.Register("actions/workflows", func(ctx context.Context, search string, offset, limit int) ([]models.LookupItem, int, error) {
		if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions") {
			return nil, 0, fmt.Errorf("%w: actions", commonauth.ErrPermissionDenied)
		}
		workflows, err := repo.ListWorkflows(ctx)
		if err != nil {
			return nil, 0, err
		}
		var items []models.LookupItem
		search = strings.ToLower(search)
		for _, wf := range workflows {
			if search != "" && !strings.Contains(strings.ToLower(wf.ID), search) && !strings.Contains(strings.ToLower(wf.Name), search) {
				continue
			}
			items = append(items, models.LookupItem{
				ID:          wf.ID,
				Name:        wf.Name,
				Description: wf.Description,
			})
		}
		total := len(items)
		if limit <= 0 {
			limit = 20
		}
		if offset >= total {
			return []models.LookupItem{}, total, nil
		}
		end := offset + limit
		if end > total {
			end = total
		}
		return items[offset:end], total, nil
	})
}
