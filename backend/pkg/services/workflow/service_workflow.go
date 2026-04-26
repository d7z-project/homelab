package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonaudit "homelab/pkg/common/audit"
	commonauth "homelab/pkg/common/auth"
	repo "homelab/pkg/repositories/workflow/actions"
	runtimepkg "homelab/pkg/runtime"
	"regexp"
	"strings"
	"time"

	"homelab/pkg/models/shared"
	workflowmodel "homelab/pkg/models/workflow"

	"github.com/expr-lang/expr"
	"github.com/google/uuid"
)

func normalizeWorkflow(workflow *workflowmodel.Workflow) {
	workflow.Meta.Name = strings.TrimSpace(workflow.Meta.Name)
	workflow.Meta.Description = strings.TrimSpace(workflow.Meta.Description)
	workflow.Meta.ServiceAccountID = strings.TrimSpace(workflow.Meta.ServiceAccountID)
	workflow.Meta.CronExpr = strings.TrimSpace(workflow.Meta.CronExpr)

	if workflow.Meta.Vars != nil {
		normalizedVars := make(map[string]workflowmodel.VarDefinition, len(workflow.Meta.Vars))
		for key, def := range workflow.Meta.Vars {
			normalizedVars[strings.TrimSpace(key)] = def
		}
		workflow.Meta.Vars = normalizedVars
	}

	for i := range workflow.Meta.Steps {
		step := &workflow.Meta.Steps[i]
		step.ID = strings.TrimSpace(step.ID)
		step.Type = strings.TrimSpace(step.Type)
		step.Name = strings.TrimSpace(step.Name)
		step.If = strings.TrimSpace(step.If)
	}
}

func validateWorkflowShape(ctx context.Context, workflow *workflowmodel.Workflow) error {
	if workflow.Meta.Name == "" {
		return errors.New("workflow name is required")
	}
	if workflow.Meta.ServiceAccountID == "" {
		return errors.New("service account is required")
	}
	if workflow.Meta.Timeout < 0 {
		return errors.New("timeout must be greater than or equal to 0")
	}
	return workflow.Meta.Validate(ctx)
}

func ValidateWorkflow(ctx context.Context, workflow *workflowmodel.Workflow) error {
	normalizeWorkflow(workflow)
	if err := validateWorkflowShape(ctx, workflow); err != nil {
		return err
	}

	// Verify ServiceAccount exists using discovery service
	registry := runtimepkg.RegistryFromContext(ctx)
	if registry == nil {
		return fmt.Errorf("registry not configured")
	}
	exists, err := registry.Verify(ctx, "rbac/serviceaccounts", workflow.Meta.ServiceAccountID)
	if err != nil {
		return fmt.Errorf("failed to verify service account: %v", err)
	}
	if !exists {
		return fmt.Errorf("service account '%s' not found", workflow.Meta.ServiceAccountID)
	}

	stepIDs := make(map[string]bool)
	// Map to store output parameters for each step for cross-reference validation
	// stepID -> map[paramName]bool
	stepOutputsMap := make(map[string]map[string]bool)

	// Validate variables
	for k, v := range workflow.Meta.Vars {
		if !workflowmodel.ActionIdRegex.MatchString(k) {
			return fmt.Errorf("invalid variable key: %s (must match %s)", k, workflowmodel.ActionIdRegex.String())
		}
		if v.RegexBackend != "" {
			if _, err := regexp.Compile(v.RegexBackend); err != nil {
				return fmt.Errorf("invalid regex for variable %s: %v", k, err)
			}
		}
	}

	for _, step := range workflow.Meta.Steps {
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

		manifestParams := make(map[string]workflowmodel.ParamDefinition)
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
						if _, ok := workflow.Meta.Vars[varKey]; !ok {
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
						if _, ok := workflow.Meta.Vars[varKey]; !ok {
							return fmt.Errorf("step %s: param %s references unknown variable %s (use '?' for optional)", step.ID, k, varKey)
						}
					}
				}
			}

			// Regex validation for static values (not containing templates)
			if !strings.Contains(v, "${{") {
				// Lookup validation
				if pDef.LookupCode != "" && v != "" {
					// Recursion check: if calling self, skip existence check as it might be a new workflow
					if pDef.LookupCode == "actions/workflows" && workflow.ID != "" && v == workflow.ID {
						// Skip discovery verify for self-reference, direct recursion check happens below
					} else {
						registry := runtimepkg.RegistryFromContext(ctx)
						if registry == nil {
							return fmt.Errorf("registry not configured")
						}
						exists, err := registry.Verify(ctx, pDef.LookupCode, v)
						if err != nil {
							return fmt.Errorf("step %s: failed to verify parameter %s via discovery code %s: %v", step.ID, k, pDef.LookupCode, err)
						}
						if !exists {
							return fmt.Errorf("step %s: parameter %s value '%s' not found in discovery code %s", step.ID, k, v, pDef.LookupCode)
						}
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

func CreateWorkflow(ctx context.Context, workflow *workflowmodel.Workflow) (*workflowmodel.Workflow, error) {
	// 强制由后端生成 ID，禁止用户指定
	workflow.ID = uuid.NewString()

	if err := ValidateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}

	if workflow.Meta.WebhookEnabled && workflow.Status.WebhookToken == "" {
		workflow.Status.WebhookToken = GenerateWebhookToken()
	}

	workflow.Status.CreatedAt = time.Now()
	workflow.Status.UpdatedAt = time.Now()
	err := repo.SaveWorkflow(ctx, workflow)

	message := fmt.Sprintf("Created workflow %s (Enabled: %v, Timeout: %ds, SA: %s, Cron: %v, Webhook: %v)", workflow.Meta.Name, workflow.Meta.Enabled, workflow.Meta.Timeout, workflow.Meta.ServiceAccountID, workflow.Meta.CronEnabled, workflow.Meta.WebhookEnabled)
	if err != nil {
		commonaudit.FromContext(ctx).Log("CreateWorkflow", workflow.ID, message, false)
		return nil, err
	}

	updated, _ := repo.GetWorkflow(ctx, workflow.ID)
	commonaudit.FromContext(ctx).Log("CreateWorkflow", workflow.ID, message, true)
	MustRuntime(ctx).TriggerManager.UpdateTriggers(*updated)
	common.NotifyCluster(ctx, common.EventWorkflowTriggerChanged, workflow.ID)
	return updated, nil
}

func UpdateWorkflow(ctx context.Context, id string, workflow *workflowmodel.Workflow) (*workflowmodel.Workflow, error) {
	// 1. Basic permission and existence check
	old, err := repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}

	// Permission check: actions/<workflow-id>
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + id) {
		return nil, fmt.Errorf("%w: actions/%s (write access required)", commonauth.ErrPermissionDenied, id)
	}

	// 2. Perform unified validation (using the new meta from request)
	// We need a temporary copy to validate
	tempWf := *old
	tempWf.Meta = workflow.Meta
	if err := ValidateWorkflow(ctx, &tempWf); err != nil {
		return nil, err
	}

	changes := []string{}
	if old.Meta.Enabled != workflow.Meta.Enabled {
		changes = append(changes, fmt.Sprintf("enabled: %v -> %v", old.Meta.Enabled, workflow.Meta.Enabled))
	}
	if old.Meta.Timeout != workflow.Meta.Timeout {
		changes = append(changes, fmt.Sprintf("timeout: %d -> %d", old.Meta.Timeout, workflow.Meta.Timeout))
	}
	if old.Meta.Name != workflow.Meta.Name {
		changes = append(changes, fmt.Sprintf("name: %s -> %s", old.Meta.Name, workflow.Meta.Name))
	}
	// ... (rest of changes logic)

	old.Meta = workflow.Meta
	old.Status.UpdatedAt = time.Now()
	if old.Meta.WebhookEnabled && old.Status.WebhookToken == "" {
		old.Status.WebhookToken = workflow.Status.WebhookToken
		if old.Status.WebhookToken == "" {
			old.Status.WebhookToken = GenerateWebhookToken()
		}
	}
	err = repo.SaveWorkflow(ctx, old)

	message := fmt.Sprintf("Updated workflow %s", workflow.Meta.Name)
	if len(changes) > 0 {
		message += ": " + strings.Join(changes, ", ")
	} else {
		message += " (no major changes)"
	}

	if err != nil {
		commonaudit.FromContext(ctx).Log("UpdateWorkflow", id, message, false)
		return nil, err
	}

	updated, _ := repo.GetWorkflow(ctx, id)
	commonaudit.FromContext(ctx).Log("UpdateWorkflow", id, message, true)
	MustRuntime(ctx).TriggerManager.UpdateTriggers(*updated)
	common.NotifyCluster(ctx, common.EventWorkflowTriggerChanged, id)
	return updated, nil
}

func ResetWebhookToken(ctx context.Context, id string) (string, error) {
	if !commonauth.PermissionsFromContext(ctx).IsAllowed("actions/" + id) {
		return "", fmt.Errorf("%w: actions/%s (write access required)", commonauth.ErrPermissionDenied, id)
	}

	workflow, err := repo.GetWorkflow(ctx, id)
	if err != nil {
		commonaudit.FromContext(ctx).Log("ResetWebhookToken", id, "Failed", false)
		return "", err
	}
	newToken := GenerateWebhookToken()
	workflow.Status.WebhookToken = newToken
	workflow.Status.UpdatedAt = time.Now()
	err = repo.SaveWorkflow(ctx, workflow)

	if err != nil {
		commonaudit.FromContext(ctx).Log("ResetWebhookToken", id, "Failed", false)
		return "", err
	}

	commonaudit.FromContext(ctx).Log("ResetWebhookToken", id, "Success", true)
	return newToken, nil
}

func GetWorkflow(ctx context.Context, id string) (*workflowmodel.Workflow, error) {
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

func GetWorkflowByWebhookToken(ctx context.Context, token string) (*workflowmodel.Workflow, error) {
	return repo.GetWorkflowByWebhookToken(ctx, token)
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
	instances, err := repo.ScanAllTaskInstancesByWorkflow(ctx, id)
	if err == nil {
		for _, inst := range instances {
			_ = repo.DeleteTaskInstance(ctx, inst.ID)
		}
		_ = RemoveWorkflowLogs(ctx, id)
	}
	MustRuntime(ctx).TriggerManager.RemoveTriggers(id)

	snapshot, _ := json.Marshal(wf)
	message := fmt.Sprintf("Deleted workflow %s. Snapshot: %s", wf.Meta.Name, string(snapshot))
	if err := repo.DeleteWorkflow(ctx, id); err != nil {
		commonaudit.FromContext(ctx).Log("DeleteWorkflow", id, message, false)
		return err
	}
	commonaudit.FromContext(ctx).Log("DeleteWorkflow", id, message, true)
	common.NotifyCluster(ctx, common.EventWorkflowTriggerChanged, id)
	return nil
}

func ScanWorkflows(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[workflowmodel.Workflow], error) {
	res, err := repo.ScanWorkflows(ctx, cursor, limit, search)
	if err != nil {
		return nil, err
	}

	perms := commonauth.PermissionsFromContext(ctx)
	if perms.IsAllowed("actions") {
		return res, nil
	}

	// Filter per-item
	var filtered []workflowmodel.Workflow
	for _, wf := range res.Items {
		if perms.IsAllowed("actions/" + wf.ID) {
			filtered = append(filtered, wf)
		}
	}
	res.Items = filtered
	return res, nil
}
