package orchestration

import (
	"context"
	"fmt"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/orchestration"
	"os"
	"regexp"
	"sync"
	"time"
)

var paramRegex = regexp.MustCompile(`\$\{\{\s*steps\.([^.]+)\.outputs\.([^ ]+)\s*\}\}`)

type Executor struct {
	runningTasks sync.Map // instanceID -> cancelFunc
}

var GlobalExecutor = &Executor{}

func (e *Executor) Execute(ctx context.Context, userID string, workflow *models.Workflow) (string, error) {
	instance := &models.TaskInstance{
		ID:         fmt.Sprintf("task_%d", time.Now().UnixNano()),
		WorkflowID: workflow.ID,
		Status:     "Running",
		UserID:     userID,
		StartedAt:  time.Now(),
		Outputs:    make(map[string]string),
	}

	workspace, err := os.MkdirTemp("", instance.ID+"_*")
	if err != nil {
		return "", err
	}
	instance.Workspace = workspace

	if err := repo.SaveTaskInstance(ctx, instance); err != nil {
		_ = os.RemoveAll(workspace)
		return "", err
	}

	taskCtx, cancel := context.WithCancel(context.Background())
	e.runningTasks.Store(instance.ID, cancel)

	logger := NewTaskLogger()

	go e.run(taskCtx, instance, workflow, logger, cancel)

	return instance.ID, nil
}

func (e *Executor) run(ctx context.Context, instance *models.TaskInstance, workflow *models.Workflow, logger *TaskLogger, cancel context.CancelFunc) {
	defer cancel()
	defer e.runningTasks.Delete(instance.ID)
	defer logger.Close()
	// NOTE: In a real system, we might want to keep the workspace for a while for debugging
	// But TODO.md says "automatically physical cleanup"
	defer func() {
		if instance.Status != "Running" {
			_ = os.RemoveAll(instance.Workspace)
		}
	}()

	logger.Logf("Starting workflow: %s (%s)", workflow.Name, workflow.ID)
	instance.Logs = logger.GetLogs()
	repo.SaveTaskInstance(context.Background(), instance)

	stepOutputs := make(map[string]map[string]string)

	for _, step := range workflow.Steps {
		logger.SetStep(step.ID)
		select {
		case <-ctx.Done():
			e.fail(instance, ctx.Err(), logger)
			instance.Status = "Cancelled"
			instance.Logs = logger.GetLogs()
			repo.SaveTaskInstance(context.Background(), instance)
			return
		default:
		}

		logger.Logf("Executing step: %s (%s)", step.Name, step.ID)

		// Resolve inputs
		inputs := make(map[string]string)
		for k, v := range step.Params {
			resolved := paramRegex.ReplaceAllStringFunc(v, func(match string) string {
				submatches := paramRegex.FindStringSubmatch(match)
				if len(submatches) < 3 {
					return match
				}
				stepID := submatches[1]
				outputKey := submatches[2]
				if outputs, ok := stepOutputs[stepID]; ok {
					if val, ok := outputs[outputKey]; ok {
						return val
					}
				}
				return match
			})
			inputs[k] = resolved
		}

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
		
		// Update logs in instance
		instance.Logs = logger.GetLogs()
		repo.SaveTaskInstance(context.Background(), instance)
	}

	// Finalize
	instance.Status = "Success"
	now := time.Now()
	instance.FinishedAt = &now
	instance.Logs = logger.GetLogs()
	repo.SaveTaskInstance(context.Background(), instance)
	logger.Log("Workflow completed successfully")
}

func (e *Executor) fail(instance *models.TaskInstance, err error, logger *TaskLogger) {
	instance.Status = "Failed"
	instance.Error = err.Error()
	now := time.Now()
	instance.FinishedAt = &now
	instance.Logs = logger.GetLogs()
	repo.SaveTaskInstance(context.Background(), instance)
	logger.Logf("Workflow failed: %v", err)
}

func (e *Executor) Cancel(instanceID string) bool {
	if cancel, ok := e.runningTasks.Load(instanceID); ok {
		cancel.(context.CancelFunc)()
		return true
	}
	return false
}
