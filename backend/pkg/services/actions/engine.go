package actions

import (
	"context"
	"fmt"
	"homelab/pkg/common"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/actions"
	"regexp"
	"runtime/debug"
	"sync"
	"time"

	"github.com/expr-lang/expr"
	"github.com/spf13/afero"
)

const (
	ActionsSubDir  = "actions"
	ProbeSubDir    = "probe"
	LogSubDir      = "logs"
	TaskPrefix     = "task_"
	DefaultTimeout = 7200 * time.Second
)

var (
	actionsFS afero.Fs
	probeFS   afero.Fs
	logFS     afero.Fs
)

// Init initializes the module-scoped virtual filesystems.
// Must be called after common.TempDir is initialized.
func Init() {
	_ = common.TempDir.MkdirAll(ActionsSubDir, 0755)
	_ = common.TempDir.MkdirAll(ProbeSubDir, 0755)
	_ = common.TempDir.MkdirAll(LogSubDir, 0755)
	actionsFS = afero.NewBasePathFs(common.TempDir, ActionsSubDir)
	probeFS = afero.NewBasePathFs(common.TempDir, ProbeSubDir)
	logFS = afero.NewBasePathFs(common.TempDir, LogSubDir)
}

var paramRegex = regexp.MustCompile(`\$\{\{\s*(?:steps\.([^.]+)\.outputs\.([^ \.?]+)|vars\.([^ \.?]+))\s*(\??)\s*\}\}`)

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
		ID:          fmt.Sprintf("%s%d", TaskPrefix, time.Now().UnixNano()),
		WorkflowID:  workflow.ID,
		Status:      "Running",
		Trigger:     trigger,
		UserID:      userID,
		Inputs:      inputs,
		StartedAt:   time.Now(),
		Outputs:     make(map[string]string),
		StepTimings: make(map[int]*models.StepTiming),
	}

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
		// Assume no timeout if specifically set to negative (though UI uses 0)
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

	logger, err := NewTaskLogger(instance.ID)
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
		// Finalization Step Index
		finalStepIdx := len(workflow.Steps) + 1

		// Record end of the step that was running before defer (if any)
		if t, ok := instance.StepTimings[instance.CurrentStep]; ok {
			if t.FinishedAt == nil {
				now := time.Now()
				t.FinishedAt = &now
			}
		}

		// Use the final step index for cleanup logs
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
			instance.CurrentStep = finalStepIdx // Only move to final step index if succeeded
			now := time.Now()
			instance.FinishedAt = &now
			logger.Log("Workflow completed")
		} else {
			// If failed/cancelled, we stay on that step index for the UI
			logger.Logf("Execution ended: %s", instance.Status)
		}

		// Record finalization timing (always happens)
		instance.StepTimings[finalStepIdx] = &models.StepTiming{StartedAt: time.Now()}
		now := time.Now()
		instance.StepTimings[finalStepIdx].FinishedAt = &now

		e.updateInstanceState(instance, logger)
		cancel()
		e.runningTasks.Delete(instance.ID)
		logger.Close()
	}()

	// Initialization Step
	instance.CurrentStep = 0
	instance.StepTimings[0] = &models.StepTiming{StartedAt: time.Now()}
	logger.SetStep(0)
	logger.Log("Initializing workflow")
	e.updateInstanceState(instance, logger)

	stepOutputs := make(map[string]map[string]string)

	for i, step := range workflow.Steps {
		// End previous step timing (Init or previous workflow step)
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
		select {
		case <-ctx.Done():
			e.fail(instance, ctx.Err(), logger)
			instance.Status = "Cancelled"
			e.updateInstanceState(instance, logger)
			return
		default:
		}

		// Resolve Step Name (No logging start here as requested to reduce noise)

		// 1. Evaluate 'if' condition
		if step.If != "" {
			shouldRun, err := e.evaluateIf(step.If, stepOutputs, instance.Inputs)
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
			inputs[k] = e.resolveVariables(v, stepOutputs, instance.Inputs)
		}

		// 3. Execute Processor
		processor, ok := GetProcessor(step.Type)
		if !ok {
			e.fail(instance, fmt.Errorf("processor not found: %s", step.Type), logger)
			return
		}

		// Validate against manifest
		manifest := processor.Manifest()
		for _, pDef := range manifest.Params {
			val := inputs[pDef.Name]
			if val == "" && !pDef.Optional {
				e.fail(instance, fmt.Errorf("missing required parameter %s for step %s", pDef.Name, step.ID), logger)
				return
			}
			if val != "" && pDef.RegexBackend != "" {
				matched, err := regexp.MatchString(pDef.RegexBackend, val)
				if err != nil {
					e.fail(instance, fmt.Errorf("invalid regex for parameter %s in step %s: %v", pDef.Name, step.ID, err), logger)
					return
				}
				if !matched {
					e.fail(instance, fmt.Errorf("parameter %s in step %s does not match required format", pDef.Name, step.ID), logger)
					return
				}
			}
		}

		taskCtx := &TaskContext{
			WorkflowID: workflow.ID,
			InstanceID: instance.ID,
			Workspace:  afero.NewBasePathFs(actionsFS, instance.Workspace),
			UserID:     instance.UserID,
			Context:    ctx,
			CancelFunc: cancel,
			Logger:     logger,
		}

		outputs, err := processor.Execute(taskCtx, inputs)
		if err != nil {
			e.fail(instance, err, logger)
			return
		}

		stepOutputs[step.ID] = outputs
		e.updateInstanceState(instance, logger)
	}
}

func (e *Executor) updateInstanceState(instance *models.TaskInstance, logger *TaskLogger) {
	_ = repo.SaveTaskInstance(context.Background(), instance)
}

func (e *Executor) resolveVariables(input string, stepOutputs map[string]map[string]string, workflowInputs map[string]string) string {
	return paramRegex.ReplaceAllStringFunc(input, func(match string) string {
		submatches := paramRegex.FindStringSubmatch(match)
		if len(submatches) < 5 {
			return match
		}

		stepID := submatches[1]
		outputKey := submatches[2]
		varKey := submatches[3]
		isOptional := submatches[4] == "?"

		var resolvedVal string
		var found bool

		if stepID != "" && outputKey != "" {
			if outputs, ok := stepOutputs[stepID]; ok {
				if val, ok := outputs[outputKey]; ok {
					resolvedVal = val
					found = true
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

func (e *Executor) evaluateIf(condition string, stepOutputs map[string]map[string]string, workflowInputs map[string]string) (bool, error) {
	resolvedIf := paramRegex.ReplaceAllStringFunc(condition, func(match string) string {
		val := e.resolveVariables(match, stepOutputs, workflowInputs)
		if val == match || val == "" {
			// If it wasn't resolved (and wasn't optionally resolved to empty string), or resolved to empty string,
			// keep it empty or return the matched string
			if val == "" {
				return `""`
			}
			return match
		}
		// Heuristic to quote strings for the expression engine
		if regexp.MustCompile(`^-?\d+(\.\d+)?$`).MatchString(val) || val == "true" || val == "false" {
			return val
		}
		return fmt.Sprintf("%q", val)
	})

	program, err := expr.Compile(resolvedIf, expr.AsBool())
	if err != nil {
		return false, err
	}

	output, err := expr.Run(program, nil)
	if err != nil {
		return false, err
	}

	return output.(bool), nil
}

func (e *Executor) fail(instance *models.TaskInstance, err error, logger *TaskLogger) {
	instance.Status = "Failed"
	instance.Error = err.Error()
	now := time.Now()
	instance.FinishedAt = &now
	e.updateInstanceState(instance, logger)
	logger.Logf("Workflow failed: %v", err)
}

func (e *Executor) Cancel(instanceID string) bool {
	if cancel, ok := e.runningTasks.Load(instanceID); ok {
		cancel.(context.CancelFunc)()
		return true
	}
	return false
}
