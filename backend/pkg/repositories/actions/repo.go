package actions

import (
	"context"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"
)

var (
	WorkflowRepo     = common.NewBaseRepository[models.WorkflowV1Meta, models.WorkflowV1Status]("actions", "workflows")
	TaskInstanceRepo = common.NewBaseRepository[models.TaskInstanceV1Meta, models.TaskInstanceV1Status]("actions", "instances")
)

// Workflow helpers

func GetWorkflow(ctx context.Context, id string) (*models.Workflow, error) {
	return WorkflowRepo.Get(ctx, id)
}

func SaveWorkflow(ctx context.Context, workflow *models.Workflow) error {
	return WorkflowRepo.Save(ctx, workflow)
}

func DeleteWorkflow(ctx context.Context, id string) error {
	return WorkflowRepo.Delete(ctx, id)
}

func ScanWorkflows(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.Workflow], error) {
	search = strings.ToLower(search)
	return WorkflowRepo.List(ctx, cursor, limit, func(wf *models.Workflow) bool {
		return search == "" || strings.Contains(strings.ToLower(wf.Meta.Name), search) || strings.Contains(strings.ToLower(wf.ID), search)
	})
}

func ScanAllWorkflows(ctx context.Context) ([]models.Workflow, error) {
	return WorkflowRepo.ListAll(ctx)
}

// TaskInstance helpers

func GetTaskInstance(ctx context.Context, id string) (*models.TaskInstance, error) {
	return TaskInstanceRepo.Get(ctx, id)
}

func SaveTaskInstance(ctx context.Context, instance *models.TaskInstance) error {
	return TaskInstanceRepo.Save(ctx, instance)
}

func DeleteTaskInstance(ctx context.Context, id string) error {
	return TaskInstanceRepo.Delete(ctx, id)
}

func ScanAllTaskInstances(ctx context.Context) ([]models.TaskInstance, error) {
	return TaskInstanceRepo.ListAll(ctx)
}

func ScanTaskInstances(ctx context.Context, cursor string, limit int, search string, workflowId string) (*models.PaginationResponse[models.TaskInstance], error) {
	search = strings.ToLower(search)
	return TaskInstanceRepo.List(ctx, cursor, limit, func(inst *models.TaskInstance) bool {
		if workflowId != "" && inst.Meta.WorkflowID != workflowId {
			return false
		}
		return search == "" || strings.Contains(strings.ToLower(inst.ID), search) || strings.Contains(strings.ToLower(inst.Meta.WorkflowID), search)
	})
}
