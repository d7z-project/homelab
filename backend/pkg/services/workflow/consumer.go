package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"gopkg.d7z.net/middleware/queue"
	"homelab/pkg/common"
	"homelab/pkg/models/shared"
	workflowmodel "homelab/pkg/models/workflow"
	repo "homelab/pkg/repositories/workflow/actions"
)

const workflowExecuteTopic = "workflow.execute"

func (rt *Runtime) StartExecutionConsumer(ctx context.Context) error {
	if rt.Deps.Queue == nil {
		return errors.New("task queue is not configured")
	}
	go rt.consumeWorkflowExecutions(rt.WithContext(ctx))
	return nil
}

func (rt *Runtime) consumeWorkflowExecutions(ctx context.Context) {
	for {
		msg, err := rt.Deps.Queue.Consume(ctx, workflowExecuteTopic, nil)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("workflow queue consume failed: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		rt.handleWorkflowExecutionMessage(ctx, msg)
	}
}

func (rt *Runtime) handleWorkflowExecutionMessage(ctx context.Context, msg *queue.Message) {
	var job workflowmodel.WorkflowExecuteJob
	if err := json.Unmarshal([]byte(msg.Body), &job); err != nil {
		log.Printf("workflow queue decode failed for message %s: %v", msg.ID, err)
		_ = msg.Ack(ctx)
		return
	}

	instance, err := repo.GetTaskInstance(ctx, job.InstanceID)
	if err != nil {
		if errors.Is(err, common.ErrNotFound) {
			_ = msg.Ack(ctx)
			return
		}
		log.Printf("workflow queue load instance %s failed: %v", job.InstanceID, err)
		_ = msg.Nack(ctx, time.Second)
		return
	}
	if instance.Status.Status != shared.TaskStatusPending {
		_ = msg.Ack(ctx)
		return
	}

	wf, err := repo.GetWorkflow(ctx, job.WorkflowID)
	if err != nil {
		if errors.Is(err, common.ErrNotFound) {
			now := time.Now()
			instance.Status.Status = shared.TaskStatusFailed
			instance.Status.Error = "workflow no longer exists"
			instance.Status.FinishedAt = &now
			_ = repo.SaveTaskInstance(ctx, instance)
			_ = msg.Ack(ctx)
			return
		}
		log.Printf("workflow queue load workflow %s failed: %v", job.WorkflowID, err)
		_ = msg.Nack(ctx, time.Second)
		return
	}

	dispatchedAt := time.Now()
	instance.Status.DispatchedAt = &dispatchedAt
	if err := repo.SaveTaskInstance(ctx, instance); err != nil {
		log.Printf("workflow queue persist dispatch %s failed: %v", job.InstanceID, err)
		_ = msg.Nack(ctx, time.Second)
		return
	}

	if _, err := rt.Executor.Execute(ctx, instance.Meta.UserID, wf, instance.Meta.Trigger, instance.Meta.Inputs, instance.ID); err != nil {
		now := time.Now()
		instance.Status.Status = shared.TaskStatusFailed
		instance.Status.Error = err.Error()
		instance.Status.FinishedAt = &now
		_ = repo.SaveTaskInstance(ctx, instance)
		_ = msg.Ack(ctx)
		return
	}
	_ = msg.Ack(ctx)
}
