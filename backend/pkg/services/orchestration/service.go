package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"homelab/pkg/models"
	repo "homelab/pkg/repositories/orchestration"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

func CreateWorkflow(ctx context.Context, workflow *models.Workflow) (*models.Workflow, error) {
	workflow.ID = uuid.New().String()
	workflow.CreatedAt = time.Now()
	workflow.UpdatedAt = time.Now()

	if err := repo.SaveWorkflow(ctx, workflow); err != nil {
		return nil, err
	}
	return workflow, nil
}

func UpdateWorkflow(ctx context.Context, id string, workflow *models.Workflow) (*models.Workflow, error) {
	old, err := repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}

	old.Name = workflow.Name
	old.Description = workflow.Description
	old.Steps = workflow.Steps
	old.UpdatedAt = time.Now()

	if err := repo.SaveWorkflow(ctx, old); err != nil {
		return nil, err
	}
	return old, nil
}

func GetWorkflow(ctx context.Context, id string) (*models.Workflow, error) {
	return repo.GetWorkflow(ctx, id)
}

func DeleteWorkflow(ctx context.Context, id string) error {
	// Cascade delete instances
	instances, err := repo.ListTaskInstances(ctx)
	if err == nil {
		for _, inst := range instances {
			if inst.WorkflowID == id {
				_ = repo.DeleteTaskInstance(ctx, inst.ID)
			}
		}
	}
	return repo.DeleteWorkflow(ctx, id)
}

func ListWorkflows(ctx context.Context) ([]models.Workflow, error) {
	return repo.ListWorkflows(ctx)
}

// Task Instance Services

func RunWorkflow(ctx context.Context, workflowID string) (string, error) {
	workflow, err := repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return "", err
	}

	authCtx := commonauth.FromContext(ctx)
	userID := "anonymous"
	if authCtx != nil {
		if authCtx.Type == "root" {
			userID = "root"
		} else {
			userID = authCtx.ID
		}
	}

	return GlobalExecutor.Execute(ctx, userID, workflow)
}

func GetTaskInstance(ctx context.Context, id string) (*models.TaskInstance, error) {
	return repo.GetTaskInstance(ctx, id)
}

func ListTaskInstances(ctx context.Context) ([]models.TaskInstance, error) {
	return repo.ListTaskInstances(ctx)
}

func CancelTaskInstance(ctx context.Context, id string) error {
	if GlobalExecutor.Cancel(id) {
		return nil
	}
	// If not running, maybe it's already finished or doesn't exist
	instance, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return err
	}
	if instance.Status == "Running" {
		instance.Status = "Cancelled"
		now := time.Now()
		instance.FinishedAt = &now
		return repo.SaveTaskInstance(ctx, instance)
	}
	return nil
}

func GetTaskLogs(ctx context.Context, id string) (string, error) {
	instance, err := repo.GetTaskInstance(ctx, id)
	if err != nil {
		return "", err
	}
	// Return logs as JSON string
	data, err := json.Marshal(instance.Logs)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

type ProbeRequest struct {
	ProcessorID string            `json:"processorId"`
	Params      map[string]string `json:"params"`
}

func (p *ProbeRequest) Bind(r *http.Request) error {
	return nil
}

func Probe(ctx context.Context, req *ProbeRequest) (map[string]string, error) {
	processor, ok := GetProcessor(req.ProcessorID)
	if !ok {
		return nil, fmt.Errorf("processor not found: %s", req.ProcessorID)
	}

	// Create a temporary workspace for probe
	workspace, err := os.MkdirTemp("", "probe_*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workspace)

	logger := NewTaskLogger()
	defer logger.Close()

	authCtx := commonauth.FromContext(ctx)
	userID := "anonymous"
	if authCtx != nil {
		if authCtx.Type == "root" {
			userID = "root"
		} else {
			userID = authCtx.ID
		}
	}

	taskCtx := &TaskContext{
		InstanceID: "probe",
		Workspace:  workspace,
		UserID:     userID,
		Context:    ctx,
		CancelFunc: func() {},
		Logger:     logger,
	}

	return processor.Execute(taskCtx, req.Params)
}
