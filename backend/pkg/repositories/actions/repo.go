package actions

import (
	"context"
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"
	"strings"

	"gopkg.d7z.net/middleware/kv"
)

// Workflow Repo

func GetWorkflow(ctx context.Context, id string) (*models.Workflow, error) {
	db := common.DB.Child("system", "actions", "workflows")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var workflow models.Workflow
	if err := json.Unmarshal([]byte(data), &workflow); err != nil {
		return nil, err
	}
	return &workflow, nil
}

func SaveWorkflow(ctx context.Context, workflow *models.Workflow) error {
	db := common.DB.Child("system", "actions", "workflows")
	data, err := json.Marshal(workflow)
	if err != nil {
		return err
	}
	return db.Put(ctx, workflow.ID, string(data), kv.TTLKeep)
}

func DeleteWorkflow(ctx context.Context, id string) error {
	_, err := common.DB.Child("system", "actions", "workflows").Delete(ctx, id)
	return err
}

func ScanWorkflows(ctx context.Context, cursor string, limit int, search string) (*models.PaginationResponse[models.Workflow], error) {
	db := common.DB.Child("system", "actions", "workflows")
	count, _ := db.Count(ctx)
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.Workflow, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var workflow models.Workflow
		if err := json.Unmarshal([]byte(v.Value), &workflow); err == nil {
			if search == "" || strings.Contains(strings.ToLower(workflow.Name), search) || strings.Contains(strings.ToLower(workflow.ID), search) {
				res = append(res, workflow)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.Workflow]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    true, // We reached the limit, there might be more
				Total:      int64(count),
			}, nil
		}
	}
	return &models.PaginationResponse[models.Workflow]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
		Total:      int64(count),
	}, nil
}

// TaskInstance Repo

func GetTaskInstance(ctx context.Context, id string) (*models.TaskInstance, error) {
	db := common.DB.Child("system", "actions", "instances")
	data, err := db.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	var instance models.TaskInstance
	if err := json.Unmarshal([]byte(data), &instance); err != nil {
		return nil, err
	}
	return &instance, nil
}

func SaveTaskInstance(ctx context.Context, instance *models.TaskInstance) error {
	db := common.DB.Child("system", "actions", "instances")
	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	return db.Put(ctx, instance.ID, string(data), kv.TTLKeep)
}

func ScanAllWorkflows(ctx context.Context) ([]models.Workflow, error) {
	db := common.DB.Child("system", "actions", "workflows")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	res := make([]models.Workflow, 0, len(items))
	for _, v := range items {
		var workflow models.Workflow
		if err := json.Unmarshal([]byte(v.Value), &workflow); err == nil {
			res = append(res, workflow)
		}
	}
	return res, nil
}

func ScanAllTaskInstances(ctx context.Context) ([]models.TaskInstance, error) {
	db := common.DB.Child("system", "actions", "instances")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	res := make([]models.TaskInstance, 0, len(items))
	for _, v := range items {
		var inst models.TaskInstance
		if err := json.Unmarshal([]byte(v.Value), &inst); err == nil {
			res = append(res, inst)
		}
	}
	return res, nil
}

func ScanTaskInstances(ctx context.Context, cursor string, limit int, search string, workflowId string) (*models.PaginationResponse[models.TaskInstance], error) {
	db := common.DB.Child("system", "actions", "instances")
	count, _ := db.Count(ctx)
	resp, err := db.ListCurrentCursor(ctx, &kv.ListOptions{
		Limit:  int64(limit * 5),
		Cursor: cursor,
	})
	if err != nil {
		return nil, err
	}

	res := make([]models.TaskInstance, 0)
	search = strings.ToLower(search)
	for _, v := range resp.Pairs {
		var instance models.TaskInstance
		if err := json.Unmarshal([]byte(v.Value), &instance); err == nil {
			if workflowId != "" && instance.WorkflowID != workflowId {
				continue
			}
			if search == "" || strings.Contains(strings.ToLower(instance.ID), search) || strings.Contains(strings.ToLower(instance.WorkflowID), search) {
				res = append(res, instance)
			}
		}
		if len(res) >= limit {
			return &models.PaginationResponse[models.TaskInstance]{
				Items:      res,
				NextCursor: v.Key,
				HasMore:    resp.HasMore || len(resp.Pairs) > 0,
				Total:      int64(count),
			}, nil
		}
	}
	return &models.PaginationResponse[models.TaskInstance]{
		Items:      res,
		NextCursor: resp.Cursor,
		HasMore:    resp.HasMore,
		Total:      int64(count),
	}, nil
}

func DeleteTaskInstance(ctx context.Context, id string) error {
	_, err := common.DB.Child("system", "actions", "instances").Delete(ctx, id)
	return err
}
