package orchestration

import (
	"context"
	"encoding/json"
	"homelab/pkg/common"
	"homelab/pkg/models"

	"gopkg.d7z.net/middleware/kv"
)

// Workflow Repo

func GetWorkflow(ctx context.Context, id string) (*models.Workflow, error) {
	db := common.DB.Child("system", "orchestration", "workflows")
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
	db := common.DB.Child("system", "orchestration", "workflows")
	data, err := json.Marshal(workflow)
	if err != nil {
		return err
	}
	return db.Put(ctx, workflow.ID, string(data), kv.TTLKeep)
}

func DeleteWorkflow(ctx context.Context, id string) error {
	_, err := common.DB.Child("system", "orchestration", "workflows").Delete(ctx, id)
	return err
}

func ListWorkflows(ctx context.Context) ([]models.Workflow, error) {
	db := common.DB.Child("system", "orchestration", "workflows")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	var res []models.Workflow
	for _, v := range items {
		var workflow models.Workflow
		if err := json.Unmarshal([]byte(v.Value), &workflow); err == nil {
			res = append(res, workflow)
		}
	}
	return res, nil
}

// TaskInstance Repo

func GetTaskInstance(ctx context.Context, id string) (*models.TaskInstance, error) {
	db := common.DB.Child("system", "orchestration", "instances")
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
	db := common.DB.Child("system", "orchestration", "instances")
	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}
	return db.Put(ctx, instance.ID, string(data), kv.TTLKeep)
}

func ListTaskInstances(ctx context.Context) ([]models.TaskInstance, error) {
	db := common.DB.Child("system", "orchestration", "instances")
	items, err := db.List(ctx, "")
	if err != nil {
		return nil, err
	}
	var res []models.TaskInstance
	for _, v := range items {
		var instance models.TaskInstance
		if err := json.Unmarshal([]byte(v.Value), &instance); err == nil {
			res = append(res, instance)
		}
	}
	return res, nil
}

func DeleteTaskInstance(ctx context.Context, id string) error {
	_, err := common.DB.Child("system", "orchestration", "instances").Delete(ctx, id)
	return err
}
