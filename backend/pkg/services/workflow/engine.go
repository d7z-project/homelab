package workflow

import (
	"context"
	"errors"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	repo "homelab/pkg/repositories/workflow/actions"
	runtimepkg "homelab/pkg/runtime"
	authservice "homelab/pkg/services/core/auth"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"homelab/pkg/models/shared"
	workflowmodel "homelab/pkg/models/workflow"

	"github.com/expr-lang/expr"
	"github.com/spf13/afero"
)

const (
	ActionsSubDir  = "actions"
	LogSubDir      = "logs"
	TaskPrefix     = "task_"
	DefaultTimeout = 7200 * time.Second
)

// Updated paramRegex to support steps.ID.status
// Groups: 1:stepID, 2:refType(outputs.KEY or status), 3:outputKey, 4:varKey, 5:isOptional(?)
var paramRegex = regexp.MustCompile(`\$\{\{\s*(?:steps\.([^.]+)\.(outputs\.([^ \.?]+)|status)|vars\.([^ \.?]+))\s*(\??)\s*\}\}`)

type Executor struct {
	runningTasks    sync.Map // instanceID -> cancelFunc
	activeWorkflows sync.Map // workflowID -> instanceID
}

func (e *Executor) Execute(ctx context.Context, userID string, workflow *workflowmodel.Workflow, trigger string, inputs map[string]string, instanceID string) (string, error) {
	rt := MustRuntime(ctx)
	// 1. Concurrency Control: Only one instance per workflow (Local Check)
	if existingInstance, loaded := e.activeWorkflows.LoadOrStore(workflow.ID, "placeholder"); loaded {
		return "", fmt.Errorf("workflow %s is already running locally (instance: %v)", workflow.ID, existingInstance)
	}

	// Double-check with DB to prevent distributed concurrent execution
	instances, err := repo.ScanAllTaskInstances(ctx)
	if err != nil {
		e.activeWorkflows.Delete(workflow.ID) // Delete the placeholder if DB query fails
		return "", fmt.Errorf("failed to check for running instances: %w", err)
	}
	for _, inst := range instances {
		if inst.Meta.WorkflowID == workflow.ID && (inst.Status.Status == "Running" || inst.Status.Status == "Pending") {
			// 健壮性：探测锁状态。如果能拿到锁，说明之前的执行者已挂
			lockKey := "action:task:" + inst.ID
			if release := rt.Deps.Locker.TryLock(ctx, lockKey); release != nil {
				// 能拿到锁，说明是僵死状态，我们手动清理并允许新任务开始
				inst.Status.Status = shared.TaskStatusFailed
				inst.Status.Error = "Interrupted by system restart or node failure"
				_ = repo.SaveTaskInstance(ctx, &inst)
				release()
				continue
			}
			e.activeWorkflows.Delete(workflow.ID)
			return "", fmt.Errorf("workflow %s is already running on another node (instance: %v)", workflow.ID, inst.ID)
		}
	}

	if instanceID == "" {
		instanceID = fmt.Sprintf("%s%d", TaskPrefix, time.Now().UnixNano())
	}

	instance := &workflowmodel.TaskInstance{
		ID: instanceID,
		Meta: workflowmodel.TaskInstanceV1Meta{
			WorkflowID:       workflow.ID,
			Trigger:          trigger,
			UserID:           userID,
			ServiceAccountID: workflow.Meta.ServiceAccountID,
			Inputs:           inputs,
			Steps:            make([]workflowmodel.Step, len(workflow.Meta.Steps)),
		},
		Status: workflowmodel.TaskInstanceV1Status{
			Status:      shared.TaskStatusRunning,
			StartedAt:   time.Now(),
			Outputs:     make(map[string]string),
			StepTimings: make(map[int]*workflowmodel.StepTiming),
		},
	}
	copy(instance.Meta.Steps, workflow.Meta.Steps)

	// Update activeWorkflows with the real instance ID
	e.activeWorkflows.Store(workflow.ID, instance.ID)

	workspace, err := afero.TempDir(rt.ActionsFS, "", instance.ID)
	if err != nil {
		e.activeWorkflows.Delete(workflow.ID)
		return "", err
	}
	instance.Status.Workspace = workspace

	if err := repo.SaveTaskInstance(ctx, instance); err != nil {
		_ = rt.ActionsFS.RemoveAll(workspace)
		e.activeWorkflows.Delete(workflow.ID)
		return "", err
	}

	// 2. Timeout logic
	timeout := DefaultTimeout
	if workflow.Meta.Timeout > 0 {
		timeout = time.Duration(workflow.Meta.Timeout) * time.Second
	} else if workflow.Meta.Timeout < 0 {
		timeout = 0
	}

	var taskCtx context.Context
	var cancel context.CancelFunc

	if timeout > 0 {
		taskCtx, cancel = context.WithTimeout(runtimepkg.DetachContext(ctx), timeout)
	} else {
		taskCtx, cancel = context.WithCancel(runtimepkg.DetachContext(ctx))
	}
	taskCtx = rt.WithContext(taskCtx)

	e.runningTasks.Store(instance.ID, cancel)

	logger, err := NewTaskLogger(taskCtx, workflow.ID, instance.ID)
	if err != nil {
		e.activeWorkflows.Delete(workflow.ID)
		cancel()
		return "", err
	}
	logger.Logf("Workspace created at: %s", workspace)

	go func() {
		defer e.activeWorkflows.Delete(workflow.ID)
		e.run(taskCtx, instance, workflow, logger, cancel)
	}()

	return instance.ID, nil
}

func (e *Executor) run(ctx context.Context, instance *workflowmodel.TaskInstance, workflow *workflowmodel.Workflow, logger *TaskLogger, cancel context.CancelFunc) {
	// 获取分布式锁，证明该节点正在处理该任务
	lockKey := "action:task:" + instance.ID
	rt := MustRuntime(ctx)
	release := rt.Deps.Locker.TryLock(ctx, lockKey)
	if release == nil {
		logger.Logf("Task %s already handled by another node", instance.ID)
		return
	}
	defer release()

	defer func() {
		finalStepIdx := len(instance.Meta.Steps) + 1
		if t, ok := instance.Status.StepTimings[instance.Status.CurrentStep]; ok {
			if t.FinishedAt == nil {
				now := time.Now()
				t.FinishedAt = &now
			}
		}
		logger.SetStep(finalStepIdx)
		if r := recover(); r != nil {
			err := fmt.Errorf("panic recovered: %v\n%s", r, string(debug.Stack()))
			e.fail(instance, err, logger)
		}
		if instance.Status.Workspace != "" {
			logger.Logf("Cleaning up workspace...")
			_ = rt.ActionsFS.RemoveAll(instance.Status.Workspace)
		}
		if instance.Status.Status == "Running" {
			instance.Status.Status = "Success"
			instance.Status.CurrentStep = finalStepIdx
			now := time.Now()
			instance.Status.FinishedAt = &now
			logger.Log("Workflow completed")
		} else {
			logger.Logf("Execution ended: %s", instance.Status.Status)
		}
		instance.Status.StepTimings[finalStepIdx] = &workflowmodel.StepTiming{StartedAt: time.Now()}
		now := time.Now()
		instance.Status.StepTimings[finalStepIdx].FinishedAt = &now
		e.updateInstanceState(instance, logger)
		cancel()
		e.runningTasks.Delete(instance.ID)
		logger.Close()
	}()

	instance.Status.CurrentStep = 0
	instance.Status.StepTimings[0] = &workflowmodel.StepTiming{StartedAt: time.Now()}
	logger.SetStep(0)
	logger.Log("Initializing workflow")
	e.updateInstanceState(instance, logger)

	// 前置校验: 确认执行身份 (ServiceAccount) 仍然存在
	if instance.Meta.ServiceAccountID != "" && instance.Meta.ServiceAccountID != "root" {
		registry := runtimepkg.RegistryFromContext(ctx)
		if registry == nil {
			e.fail(instance, fmt.Errorf("registry not configured"), logger)
			return
		}
		saCtx := commonauth.WithRoot(runtimepkg.DetachContext(ctx))
		saExists, err := registry.Verify(saCtx, "rbac/serviceaccounts", instance.Meta.ServiceAccountID)
		if err != nil || !saExists {
			e.fail(instance, fmt.Errorf("service account '%s' no longer exists, workflow cannot execute", instance.Meta.ServiceAccountID), logger)
			return
		}
	}

	stepOutputs := make(map[string]map[string]string)
	stepStatuses := make(map[string]bool)

	for i, step := range instance.Meta.Steps {
		if t, ok := instance.Status.StepTimings[instance.Status.CurrentStep]; ok {
			if t.FinishedAt == nil {
				now := time.Now()
				t.FinishedAt = &now
			}
		}

		instance.Status.CurrentStep = i + 1
		instance.Status.StepTimings[instance.Status.CurrentStep] = &workflowmodel.StepTiming{StartedAt: time.Now()}
		logger.SetStep(instance.Status.CurrentStep)
		e.updateInstanceState(instance, logger)

		// Create impersonated context for this step
		// This ensures all repo calls and processor logic respect the SA permissions
		identityType := "sa"
		identityID := instance.Meta.ServiceAccountID
		if identityID == "" || identityID == "root" {
			identityType = "root"
			identityID = ""
		}
		perms, err := authservice.GetPermissions(ctx, instance.Meta.ServiceAccountID, "*", "*")
		if err != nil {
			e.fail(instance, fmt.Errorf("failed to load workflow execution permissions: %w", err), logger)
			return
		}
		impersonatedCtx := commonauth.WithIdentity(ctx, &commonauth.AuthContext{
			ID:   identityID,
			Type: identityType,
		}, perms)

		select {
		case <-ctx.Done():
			e.fail(instance, ctx.Err(), logger)
			return
		default:
		}

		// 1. Evaluate 'if' condition
		if step.If != "" {
			shouldRun, err := e.evaluateIf(step.If, stepOutputs, stepStatuses, instance.Meta.Inputs)
			if err != nil {
				e.fail(instance, fmt.Errorf("invalid if condition in step %s: %v", step.ID, err), logger)
				return
			}
			if !shouldRun {
				logger.Logf("Step skipped")
				e.updateInstanceState(instance, logger)
				continue
			}
		}

		// 2. Resolve inputs
		inputs := make(map[string]string)
		for k, v := range step.Params {
			inputs[k] = e.resolveVariables(v, stepOutputs, stepStatuses, instance.Meta.Inputs)
		}

		// 3. Execute Processor
		processor, ok := GetProcessor(step.Type)
		if !ok {
			e.fail(instance, fmt.Errorf("processor not found: %s", step.Type), logger)
			return
		}

		// Validate against manifest
		manifest := processor.Manifest()
		var validationErr error
		for _, pDef := range manifest.Params {
			val := inputs[pDef.Name]
			if val == "" && !pDef.Optional {
				validationErr = fmt.Errorf("missing required parameter %s for step %s", pDef.Name, step.ID)
				break
			}
			if val != "" && pDef.RegexBackend != "" {
				matched, err := regexp.MatchString(pDef.RegexBackend, val)
				if err != nil {
					validationErr = fmt.Errorf("invalid regex for parameter %s in step %s: %v", pDef.Name, step.ID, err)
					break
				}
				if !matched {
					validationErr = fmt.Errorf("parameter %s in step %s does not match required format", pDef.Name, step.ID)
					break
				}
			}
		}

		if validationErr != nil {
			if step.Fail {
				logger.Logf("Step validation failed, but allow error (fail:true) is set: %v", validationErr)
				stepStatuses[step.ID] = false
				e.updateInstanceState(instance, logger)
				continue
			}
			e.fail(instance, validationErr, logger)
			return
		}

		taskCtx := &TaskContext{
			WorkflowID:       workflow.ID,
			InstanceID:       instance.ID,
			Workspace:        afero.NewBasePathFs(rt.ActionsFS, instance.Status.Workspace),
			UserID:           instance.Meta.UserID,
			ServiceAccountID: workflow.Meta.ServiceAccountID,
			Context:          impersonatedCtx, // Use impersonated context
			CancelFunc:       cancel,
			Logger:           logger,
		}

		outputs, err := processor.Execute(taskCtx, inputs)
		if err != nil {
			if step.Fail {
				logger.Logf("Step failed, but allow error (fail:true) is set: %v", err)
				stepStatuses[step.ID] = false
				e.updateInstanceState(instance, logger)
				continue
			}
			e.fail(instance, err, logger)
			return
		}

		stepOutputs[step.ID] = outputs
		stepStatuses[step.ID] = true
		e.updateInstanceState(instance, logger)
	}
}

func (e *Executor) updateInstanceState(instance *workflowmodel.TaskInstance, logger *TaskLogger) {
	if instance == nil {
		return
	}
	if repo.StorageReady(logger.Context()) {
		_ = repo.SaveTaskInstance(runtimepkg.DetachContext(logger.Context()), instance)
	}
}

func (e *Executor) resolveVariables(input string, stepOutputs map[string]map[string]string, stepStatuses map[string]bool, workflowInputs map[string]string) string {
	return paramRegex.ReplaceAllStringFunc(input, func(match string) string {
		submatches := paramRegex.FindStringSubmatch(match)
		if len(submatches) < 6 {
			return match
		}

		stepID := submatches[1]
		refType := submatches[2] // "outputs.KEY" or "status"
		outputKey := submatches[3]
		varKey := submatches[4]
		isOptional := submatches[5] == "?"

		var resolvedVal string
		var found bool

		if stepID != "" {
			if refType == "status" {
				if status, ok := stepStatuses[stepID]; ok {
					if status {
						resolvedVal = "true"
					} else {
						resolvedVal = "false"
					}
					found = true
				}
			} else if outputKey != "" {
				if outputs, ok := stepOutputs[stepID]; ok {
					if val, ok := outputs[outputKey]; ok {
						resolvedVal = val
						found = true
					}
				}
			}
		} else if varKey != "" {
			if val, ok := workflowInputs[varKey]; ok {
				resolvedVal = val
				found = true
			}
		}

		if found {
			return resolvedVal
		}

		if isOptional {
			return ""
		}
		return match
	})
}

func (e *Executor) evaluateIf(condition string, stepOutputs map[string]map[string]string, stepStatuses map[string]bool, workflowInputs map[string]string) (bool, error) {
	matches := paramRegex.FindAllStringSubmatch(condition, -1)
	env := make(map[string]interface{})
	exprStr := condition

	for i, match := range matches {
		if len(match) < 6 {
			continue
		}
		fullMatch := match[0]
		stepID := match[1]
		refType := match[2]
		outputKey := match[3]
		varKey := match[4]
		isOptional := match[5] == "?"

		var resolvedVal interface{}
		var found bool

		if stepID != "" {
			if refType == "status" {
				if status, ok := stepStatuses[stepID]; ok {
					resolvedVal = status
					found = true
				}
			} else if outputKey != "" {
				if outputs, ok := stepOutputs[stepID]; ok {
					if val, ok := outputs[outputKey]; ok {
						resolvedVal = val
						found = true
					}
				}
			}
		} else if varKey != "" {
			if val, ok := workflowInputs[varKey]; ok {
				resolvedVal = val
				found = true
			}
		}

		if !found {
			if isOptional {
				resolvedVal = ""
			} else {
				resolvedVal = nil
			}
		}

		placeholder := fmt.Sprintf("__v%d", i)
		exprStr = strings.Replace(exprStr, fullMatch, placeholder, 1)
		env[placeholder] = resolvedVal
	}

	program, err := expr.Compile(exprStr, expr.Env(env), expr.AsBool())
	if err != nil {
		return false, err
	}

	output, err := expr.Run(program, env)
	if err != nil {
		return false, err
	}

	return output.(bool), nil
}

func (e *Executor) fail(instance *workflowmodel.TaskInstance, err error, logger *TaskLogger) {
	if errors.Is(err, context.Canceled) {
		instance.Status.Status = "Cancelled"
	} else {
		instance.Status.Status = "Failed"
	}
	instance.Status.Error = err.Error()
	now := time.Now()
	instance.Status.FinishedAt = &now
	e.updateInstanceState(instance, logger)
	logger.Logf("Workflow %s: %v", instance.Status.Status, err)
}

func (e *Executor) Cancel(instanceID string) bool {
	if cancel, ok := e.runningTasks.Load(instanceID); ok {
		cancel.(context.CancelFunc)()
		return true
	}
	return false
}
