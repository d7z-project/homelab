package actions

import (
	"context"
	"strconv"
	"strings"
	"time"

	"homelab/pkg/common"
	"homelab/pkg/models/shared"
	workflowmodel "homelab/pkg/models/workflow"

	"gopkg.d7z.net/middleware/kv"
)

var (
	workflowRepo     *common.ResourceRepository[workflowmodel.WorkflowV1Meta, workflowmodel.WorkflowV1Status]
	taskInstanceRepo *common.ResourceRepository[workflowmodel.TaskInstanceV1Meta, workflowmodel.TaskInstanceV1Status]
	logDB            kv.KV
)

func Configure(db kv.KV) {
	workflowRepo = common.NewResourceRepository[workflowmodel.WorkflowV1Meta, workflowmodel.WorkflowV1Status](db, "actions", "workflows")
	taskInstanceRepo = common.NewResourceRepository[workflowmodel.TaskInstanceV1Meta, workflowmodel.TaskInstanceV1Status](db, "actions", "instances")
	logDB = db
}

// Workflow helpers

func GetWorkflow(ctx context.Context, id string) (*workflowmodel.Workflow, error) {
	return workflowRepo.Get(ctx, id)
}

func SaveWorkflow(ctx context.Context, workflow *workflowmodel.Workflow) error {
	return workflowRepo.Save(ctx, workflow)
}

func UpdateWorkflowStatus(ctx context.Context, id string, apply func(*workflowmodel.WorkflowV1Status)) error {
	return workflowRepo.UpdateStatus(ctx, id, apply)
}

func DeleteWorkflow(ctx context.Context, id string) error {
	return workflowRepo.Delete(ctx, id)
}

func ScanWorkflows(ctx context.Context, cursor string, limit int, search string) (*shared.PaginationResponse[workflowmodel.Workflow], error) {
	search = strings.ToLower(search)
	return workflowRepo.List(ctx, cursor, limit, func(wf *workflowmodel.Workflow) bool {
		return search == "" || strings.Contains(strings.ToLower(wf.Meta.Name), search) || strings.Contains(strings.ToLower(wf.ID), search)
	})
}

func WorkflowUsesServiceAccount(ctx context.Context, serviceAccountID string) (bool, *workflowmodel.Workflow, error) {
	items, err := workflowRepo.ListAllFiltered(ctx, func(wf *workflowmodel.Workflow) bool {
		return wf.Meta.ServiceAccountID == serviceAccountID
	})
	if err != nil {
		return false, nil, err
	}
	if len(items) == 0 {
		return false, nil, nil
	}
	return true, &items[0], nil
}

// TaskInstance helpers

func GetTaskInstance(ctx context.Context, id string) (*workflowmodel.TaskInstance, error) {
	return taskInstanceRepo.Get(ctx, id)
}

func SaveTaskInstance(ctx context.Context, instance *workflowmodel.TaskInstance) error {
	return taskInstanceRepo.Save(ctx, instance)
}

func DeleteTaskInstance(ctx context.Context, id string) error {
	return taskInstanceRepo.Delete(ctx, id)
}

func ScanAllTaskInstances(ctx context.Context) ([]workflowmodel.TaskInstance, error) {
	return taskInstanceRepo.ListAll(ctx)
}

func ScanAllTaskInstancesByWorkflow(ctx context.Context, workflowID string) ([]workflowmodel.TaskInstance, error) {
	return taskInstanceRepo.ListAllFiltered(ctx, func(inst *workflowmodel.TaskInstance) bool {
		return inst.Meta.WorkflowID == workflowID
	})
}

func ScanTaskInstances(ctx context.Context, cursor string, limit int, search string, workflowId string) (*shared.PaginationResponse[workflowmodel.TaskInstance], error) {
	search = strings.ToLower(search)
	return taskInstanceRepo.List(ctx, cursor, limit, func(inst *workflowmodel.TaskInstance) bool {
		if workflowId != "" && inst.Meta.WorkflowID != workflowId {
			return false
		}
		return search == "" || strings.Contains(strings.ToLower(inst.ID), search) || strings.Contains(strings.ToLower(inst.Meta.WorkflowID), search)
	})
}

func StorageReady(ctx context.Context) bool {
	return logDB != nil
}

func AppendTaskLogLine(ctx context.Context, instanceID string, stepIndex int, key string, line string, ttl time.Duration) error {
	if logDB == nil {
		return nil
	}
	return logDB.Child("system", "task:logs", instanceID, strconv.Itoa(stepIndex)).Put(ctx, key, line, ttl)
}

func DeleteTaskLogs(ctx context.Context, instanceID string) error {
	if logDB == nil {
		return nil
	}
	return logDB.Child("system", "task:logs", instanceID).DeleteAll(ctx)
}
