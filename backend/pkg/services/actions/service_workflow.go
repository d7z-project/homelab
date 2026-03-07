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
	"regexp"
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
