package actions

import (
	"context"
	"homelab/pkg/common"
	"strings"

	"homelab/pkg/models/shared"
	workflowmodel "homelab/pkg/models/workflow"
)

var (
	WorkflowRepo     = common.NewBaseRepository[workflowmodel.WorkflowV1Meta, workflowmodel.WorkflowV1Status]("actions", "workflows")
	TaskInstanceRepo = common.NewBaseRepository[workflowmodel.TaskInstanceV1Meta, workflowmodel.TaskInstanceV1Status]("actions", "instances")
)

// Workflow helpers

func GetWorkflow(ctx context.Context, id string) (*workflowmodel.Workflow, error) {
	return WorkflowRepo.Get(ctx, id)
}

func SaveWorkflow(ctx context.Context, workflow *workflowmodel.Workflow) error {
	return WorkflowRepo.Save(ctx, workflow)
}

func DeleteWorkflow(ctx context.Context, id string) error {
	return WorkflowRepo.Delete(ctx, id)
}

func ScanWorkflows(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[workflowmodel.Workflow], error) {
	search = strings.ToLower(search)
	return WorkflowRepo.List(ctx, cursor, limit, func(wf *workflowmodel.Workflow) bool {
		return search == "" || strings.Contains(strings.ToLower(wf.Meta.Name), search) || strings.Contains(strings.ToLower(wf.ID), search)
	})
}

func ScanAllWorkflows(ctx context.Context) ([]workflowmodel.Workflow, error) {
	return WorkflowRepo.ListAll(ctx)
}

// TaskInstance helpers

func GetTaskInstance(ctx context.Context, id string) (*workflowmodel.TaskInstance, error) {
	return TaskInstanceRepo.Get(ctx, id)
}

func SaveTaskInstance(ctx context.Context, instance *workflowmodel.TaskInstance) error {
	return TaskInstanceRepo.Save(ctx, instance)
}

func DeleteTaskInstance(ctx context.Context, id string) error {
	return TaskInstanceRepo.Delete(ctx, id)
}

func ScanAllTaskInstances(ctx context.Context) ([]workflowmodel.TaskInstance, error) {
	return TaskInstanceRepo.ListAll(ctx)
}

func ScanTaskInstances(ctx context.Context, cursor string, limit int, search string, workflowId string) (*shared.PaginationResponse[workflowmodel.TaskInstance], error) {
	search = strings.ToLower(search)
	return TaskInstanceRepo.List(ctx, cursor, limit, func(inst *workflowmodel.TaskInstance) bool {
		if workflowId != "" && inst.Meta.WorkflowID != workflowId {
			return false
		}
		return search == "" || strings.Contains(strings.ToLower(inst.ID), search) || strings.Contains(strings.ToLower(inst.Meta.WorkflowID), search)
	})
}
