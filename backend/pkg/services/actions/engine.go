package actions

import (
	"context"
	"errors"
	"fmt"
	"homelab/pkg/common"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/actions"
	"regexp"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/spf13/afero"
)

const (
	ActionsSubDir  = "actions"
	LogSubDir      = "logs"
	TaskPrefix     = "task_"
	DefaultTimeout = 7200 * time.Second
)

var (
	actionsFS afero.Fs
	logFS     afero.Fs
)

// Init initializes the module-scoped virtual filesystems.
// Must be called after common.FS and common.TempDir are initialized.
func Init() {
	_ = common.TempDir.MkdirAll(ActionsSubDir, 0755)
	_ = common.FS.MkdirAll(LogSubDir, 0755)
	actionsFS = afero.NewBasePathFs(common.TempDir, ActionsSubDir)
	logFS = afero.NewBasePathFs(common.FS, LogSubDir)
}

// Updated paramRegex to support steps.ID.status
// Groups: 1:stepID, 2:refType(outputs.KEY or status), 3:outputKey, 4:varKey, 5:isOptional(?)
var paramRegex = regexp.MustCompile(`\$\{\{\s*(?:steps\.([^.]+)\.(outputs\.([^ \.?]+)|status)|vars\.([^ \.?]+))\s*(\??)\s*\}\}`)

type Executor struct {
	runningTasks    sync.Map // instanceID -> cancelFunc
	activeWorkflows sync.Map // workflowID -> instanceID
}

var GlobalExecutor = &Executor{}

func (e *Executor) Execute(ctx context.Context, userID string, workflow *models.Workflow, trigger string, inputs map[string]string) (string, error) {
	// 1. Concurrency Control: Only one instance per workflow
	if existingInstance, loaded := e.activeWorkflows.LoadOrStore(workflow.ID, "placeholder"); loaded {
		return "", fmt.Errorf("workflow %s is already running (instance: %v)", workflow.ID, existingInstance)
	}

	instance := &models.TaskInstance{
		ID:               fmt.Sprintf("%s%d", TaskPrefix, time.Now().UnixNano()),
		WorkflowID:       workflow.ID,
		Status:           "Running",
		Trigger:          trigger,
		UserID:           userID,
		ServiceAccountID: workflow.ServiceAccountID,
		Inputs:           inputs,
		StartedAt:        time.Now(),
		Outputs:          make(map[string]string),
		Steps:            make([]models.Step, len(workflow.Steps)),
		StepTimings:      make(map[int]*models.StepTiming),
	}
	copy(instance.Steps, workflow.Steps)

	// Update activeWorkflows with the real instance ID
	e.activeWorkflows.Store(workflow.ID, instance.ID)

	workspace, err := afero.TempDir(actionsFS, "", instance.ID)
	if err != nil {
		e.activeWorkflows.Delete(workflow.ID)
		return "", err
	}
	instance.Workspace = workspace

	if err := repo.SaveTaskInstance(ctx, instance); err != nil {
		_ = actionsFS.RemoveAll(workspace)
		e.activeWorkflows.Delete(workflow.ID)
		return "", err
	}

	// 2. Timeout logic
	timeout := DefaultTimeout
	if workflow.Timeout > 0 {
		timeout = time.Duration(workflow.Timeout) * time.Second
	} else if workflow.Timeout < 0 {
		timeout = 0
	}

	var taskCtx context.Context
	var cancel context.CancelFunc

	if timeout > 0 {
		taskCtx, cancel = context.WithTimeout(context.Background(), timeout)
	} else {
		taskCtx, cancel = context.WithCancel(context.Background())
	}

	e.runningTasks.Store(instance.ID, cancel)

	logger, err := NewTaskLogger(workflow.ID, instance.ID)
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

func (e *Executor) run(ctx context.Context, instance *models.TaskInstance, workflow *models.Workflow, logger *TaskLogger, cancel context.CancelFunc) {
	defer func() {
		finalStepIdx := len(instance.Steps) + 1
		if t, ok := instance.StepTimings[instance.CurrentStep]; ok {
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
		if instance.Workspace != "" {
			logger.Logf("Cleaning up workspace...")
			_ = actionsFS.RemoveAll(instance.Workspace)
		}
		if instance.Status == "Running" {
			instance.Status = "Success"
			instance.CurrentStep = finalStepIdx
			now := time.Now()
			instance.FinishedAt = &now
			logger.Log("Workflow completed")
		} else {
			logger.Logf("Execution ended: %s", instance.Status)
		}
		instance.StepTimings[finalStepIdx] = &models.StepTiming{StartedAt: time.Now()}
		now := time.Now()
		instance.StepTimings[finalStepIdx].FinishedAt = &now
		e.updateInstanceState(instance, logger)
		cancel()
		e.runningTasks.Delete(instance.ID)
		logger.Close()
	}()

	instance.CurrentStep = 0
	instance.StepTimings[0] = &models.StepTiming{StartedAt: time.Now()}
	logger.SetStep(0)
	logger.Log("Initializing workflow")
	e.updateInstanceState(instance, logger)

	stepOutputs := make(map[string]map[string]string)
	stepStatuses := make(map[string]bool)

	for i, step := range instance.Steps {
		if t, ok := instance.StepTimings[instance.CurrentStep]; ok {
			if t.FinishedAt == nil {
				now := time.Now()
				t.FinishedAt = &now
			}
		}

		instance.CurrentStep = i + 1
		instance.StepTimings[instance.CurrentStep] = &models.StepTiming{StartedAt: time.Now()}
		logger.SetStep(instance.CurrentStep)
		e.updateInstanceState(instance, logger)

		// Create impersonated context for this step
		// This ensures all repo calls and processor logic respect the SA permissions
		impersonatedCtx := commonauth.WithAuth(ctx, &commonauth.AuthContext{
			ID:   instance.ServiceAccountID,
			Type: "sa",
		})

		select {
		case <-ctx.Done():
			e.fail(instance, ctx.Err(), logger)
			return
		default:
		}

		// 1. Evaluate 'if' condition
		if step.If != "" {
			shouldRun, err := e.evaluateIf(step.If, stepOutputs, stepStatuses, instance.Inputs)
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
			inputs[k] = e.resolveVariables(v, stepOutputs, stepStatuses, instance.Inputs)
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
			Workspace:        afero.NewBasePathFs(actionsFS, instance.Workspace),
			UserID:           instance.UserID,
			ServiceAccountID: workflow.ServiceAccountID,
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

func (e *Executor) updateInstanceState(instance *models.TaskInstance, logger *TaskLogger) {
	if instance == nil {
		return
	}
	if common.DB != nil {
		_ = repo.SaveTaskInstance(context.Background(), instance)
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

func (e *Executor) fail(instance *models.TaskInstance, err error, logger *TaskLogger) {
	if errors.Is(err, context.Canceled) {
		instance.Status = "Cancelled"
	} else {
		instance.Status = "Failed"
	}
	instance.Error = err.Error()
	now := time.Now()
	instance.FinishedAt = &now
	e.updateInstanceState(instance, logger)
	logger.Logf("Workflow %s: %v", instance.Status, err)
}

func (e *Executor) Cancel(instanceID string) bool {
	if cancel, ok := e.runningTasks.Load(instanceID); ok {
		cancel.(context.CancelFunc)()
		return true
	}
	return false
}
