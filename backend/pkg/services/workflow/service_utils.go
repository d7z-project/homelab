package workflow

import (
	"context"
	"fmt"
	commonauth "homelab/pkg/common/auth"
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

func Probe(ctx context.Context, processorID string, params map[string]string) (map[string]string, error) {
	rt := MustRuntime(ctx)
	processor, ok := GetProcessor(processorID)
	if !ok {
		return nil, fmt.Errorf("processor not found: %s", processorID)
	}

	// Generate a unique ID for this probe to avoid log collision
	instanceID := fmt.Sprintf("probe_%d", time.Now().UnixNano())

	// Create a temporary workspace for probe in actionsFS
	workspace, err := afero.TempDir(rt.ActionsFS, "", instanceID)
	if err != nil {
		return nil, err
	}
	defer rt.ActionsFS.RemoveAll(workspace)

	// Use '_probe' as a reserved workflow ID for system-level tests
	logger, err := NewTaskLogger(ctx, "_probe", instanceID)
	if err != nil {
		return nil, err
	}
	defer logger.Close()
	defer RemoveTaskLogs(ctx, "_probe", instanceID)

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
		Workspace:        afero.NewBasePathFs(rt.ActionsFS, workspace),
		UserID:           userID,
		ServiceAccountID: "root", // Probes run as root for full validation
		Context:          ctx,
		CancelFunc:       func() {},
		Logger:           logger,
	}

	return processor.Execute(taskCtx, params)
}
