package orchestration

import (
	"context"
	"fmt"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/orchestration"
	"os"
	"regexp"
	"runtime/debug"
	"sync"
	"time"

	"github.com/expr-lang/expr"
)

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
		ID:         fmt.Sprintf("task_%d", time.Now().UnixNano()),
		WorkflowID: workflow.ID,
		Status:     "Running",
		Trigger:    trigger,
		UserID:     userID,
		Inputs:     inputs,
		StartedAt:  time.Now(),
		Outputs:    make(map[string]string),
	}

	// Update activeWorkflows with the real instance ID
	e.activeWorkflows.Store(workflow.ID, instance.ID)

	workspace, err := os.MkdirTemp("", instance.ID+"_*")
	if err != nil {
		e.activeWorkflows.Delete(workflow.ID)
		return "", err
	}
	instance.Workspace = workspace

	if err := repo.SaveTaskInstance(ctx, instance); err != nil {
		_ = os.RemoveAll(workspace)
		e.activeWorkflows.Delete(workflow.ID)
		return "", err
	}

	// 2. Timeout logic
	timeout := 7200 * time.Second // Default 2h
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

	logger := NewTaskLogger()

	go func() {
		defer e.activeWorkflows.Delete(workflow.ID)
		e.run(taskCtx, instance, workflow, logger, cancel)
	}()

	return instance.ID, nil
}

func (e *Executor) run(ctx context.Context, instance *models.TaskInstance, workflow *models.Workflow, logger *TaskLogger, cancel context.CancelFunc) {
	defer func() {
		if r := recover(); r != nil {
			err := fmt.Errorf("panic recovered: %v\n%s", r, string(debug.Stack()))
			e.fail(instance, err, logger)
		}
	}()
	defer cancel()
	defer e.runningTasks.Delete(instance.ID)
	defer logger.Close()
	defer func() {
		if instance.Status != "Running" {
			_ = os.RemoveAll(instance.Workspace)
		}
	}()

	logger.Logf("Starting workflow: %s (%s)", workflow.Name, workflow.ID)
	e.updateInstanceState(instance, logger)

	stepOutputs := make(map[string]map[string]string)

	for _, step := range workflow.Steps {
		logger.SetStep(step.ID)
		select {
		case <-ctx.Done():
			e.fail(instance, ctx.Err(), logger)
			instance.Status = "Cancelled"
			e.updateInstanceState(instance, logger)
			return
		default:
		}

		// Resolve Step Name
		resolvedName := e.resolveVariables(step.Name, stepOutputs, instance.Inputs)
		logger.Logf("Executing step: %s (%s)", resolvedName, step.ID)

		// 1. Evaluate 'if' condition
		if step.If != "" {
			shouldRun, err := e.evaluateIf(step.If, stepOutputs, instance.Inputs)
			if err != nil {
				e.fail(instance, fmt.Errorf("invalid if condition in step %s: %v", step.ID, err), logger)
				return
			}
			if !shouldRun {
				logger.Logf("Step skipped (if condition evaluated to false)")
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

		taskCtx := &TaskContext{
			InstanceID: instance.ID,
			Workspace:  instance.Workspace,
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
		logger.Logf("Step %s completed successfully", step.ID)
		e.updateInstanceState(instance, logger)
	}

	// Finalize
	instance.Status = "Success"
	now := time.Now()
	instance.FinishedAt = &now
	e.updateInstanceState(instance, logger)
	logger.Log("Workflow completed successfully")
}

func (e *Executor) updateInstanceState(instance *models.TaskInstance, logger *TaskLogger) {
	instance.Logs = logger.GetLogs()
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
