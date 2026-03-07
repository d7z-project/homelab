package actions

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
	"net/http"
	"regexp"
	"time"

	"github.com/spf13/afero"
)

func ValidateRegex(regex string) error {
	if regex == "" {
		return nil
	}
	_, err := regexp.Compile(regex)
	return err
}

type ProbeRequest struct {
	ProcessorID string            `json:"processorId"`
	Params      map[string]string `json:"params"`
}

func (p *ProbeRequest) Bind(r *http.Request) error {
	if p.ProcessorID == "" {
		return fmt.Errorf("processorId is required")
	}
	return nil
}

func Probe(ctx context.Context, req *ProbeRequest) (map[string]string, error) {
	processor, ok := GetProcessor(req.ProcessorID)
	if !ok {
		return nil, fmt.Errorf("processor not found: %s", req.ProcessorID)
	}

	// Generate a unique ID for this probe to avoid log collision
	instanceID := fmt.Sprintf("probe_%d", time.Now().UnixNano())

	// Create a temporary workspace for probe in actionsFS
	workspace, err := afero.TempDir(actionsFS, "", instanceID)
	if err != nil {
		return nil, err
	}
	defer actionsFS.RemoveAll(workspace)

	// Use '_probe' as a reserved workflow ID for system-level tests
	logger, err := NewTaskLogger("_probe", instanceID)
	if err != nil {
		return nil, err
	}
	defer logger.Close()
	defer RemoveTaskLogs("_probe", instanceID)

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
		WorkflowID:       "_probe",
		InstanceID:       instanceID,
		Workspace:        afero.NewBasePathFs(actionsFS, workspace),
		UserID:           userID,
		ServiceAccountID: "root", // Probes run as root for full validation
		Context:          ctx,
		CancelFunc:       func() {},
		Logger:           logger,
	}

	return processor.Execute(taskCtx, req.Params)
}
