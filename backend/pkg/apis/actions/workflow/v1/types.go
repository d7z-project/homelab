package v1

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var actionIDRegex = regexp.MustCompile(`^[a-z0-9_]+$`)
var resourceIDRegex = regexp.MustCompile(`^[a-z0-9_\-]+$`)

type VarDefinition struct {
	Description   string `json:"description"`
	Default       string `json:"default"`
	Required      bool   `json:"required"`
	RegexFrontend string `json:"regexFrontend"`
	RegexBackend  string `json:"regexBackend"`
}

type WorkflowMeta struct {
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	Enabled          bool                     `json:"enabled"`
	Timeout          int                      `json:"timeout"`
	ServiceAccountID string                   `json:"serviceAccountId"`
	CronEnabled      bool                     `json:"cronEnabled"`
	CronExpr         string                   `json:"cronExpr"`
	WebhookEnabled   bool                     `json:"webhookEnabled"`
	Vars             map[string]VarDefinition `json:"vars"`
	Steps            []Step                   `json:"steps"`
}

type WorkflowStatus struct {
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	WebhookToken string    `json:"webhookToken,omitempty"`
}

type Workflow struct {
	ID              string         `json:"id"`
	Meta            WorkflowMeta   `json:"meta"`
	Status          WorkflowStatus `json:"status"`
	Generation      int64          `json:"generation"`
	ResourceVersion int64          `json:"resourceVersion"`
}

func (w *Workflow) Bind(_ *http.Request) error {
	if w.ID != "" && !resourceIDRegex.MatchString(w.ID) {
		return errors.New("invalid id format, only lowercase letters, numbers, underscores and hyphens are allowed")
	}
	w.Meta.Name = strings.TrimSpace(w.Meta.Name)
	w.Meta.Description = strings.TrimSpace(w.Meta.Description)
	w.Meta.ServiceAccountID = strings.TrimSpace(w.Meta.ServiceAccountID)
	w.Meta.CronExpr = strings.TrimSpace(w.Meta.CronExpr)
	if w.Meta.Name == "" {
		return errors.New("workflow name is required")
	}
	if w.Meta.ServiceAccountID == "" {
		return errors.New("service account is required")
	}
	if w.Meta.Timeout < 0 {
		return errors.New("timeout must be greater than or equal to 0")
	}
	if len(w.Meta.Steps) == 0 {
		return errors.New("at least one step is required")
	}
	for key, def := range w.Meta.Vars {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return errors.New("variable key is required")
		}
		if !actionIDRegex.MatchString(trimmedKey) {
			return fmt.Errorf("invalid variable key '%s': only lowercase letters, numbers and underscores are allowed", trimmedKey)
		}
		if trimmedKey != key {
			delete(w.Meta.Vars, key)
			w.Meta.Vars[trimmedKey] = def
		}
	}
	for i := range w.Meta.Steps {
		step := &w.Meta.Steps[i]
		step.ID = strings.TrimSpace(step.ID)
		step.Type = strings.TrimSpace(step.Type)
		step.Name = strings.TrimSpace(step.Name)
		step.If = strings.TrimSpace(step.If)
		if step.ID == "" {
			return fmt.Errorf("step %d: ID is required", i+1)
		}
		if !actionIDRegex.MatchString(step.ID) {
			return fmt.Errorf("step %d: invalid ID '%s': only lowercase letters, numbers and underscores are allowed", i+1, step.ID)
		}
		if step.Type == "" {
			return fmt.Errorf("step %d (%s): type is required", i+1, step.ID)
		}
	}
	return nil
}

type Step struct {
	ID     string            `json:"id"`
	Type   string            `json:"type"`
	Name   string            `json:"name"`
	If     string            `json:"if"`
	Params map[string]string `json:"params"`
	Fail   bool              `json:"fail"`
}

type StepTiming struct {
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

type TaskInstanceMeta struct {
	WorkflowID       string            `json:"workflowId"`
	Trigger          string            `json:"trigger"`
	UserID           string            `json:"userId"`
	ServiceAccountID string            `json:"serviceAccountId"`
	Inputs           map[string]string `json:"inputs"`
	Steps            []Step            `json:"steps"`
}

type TaskInstanceStatus struct {
	Status      string              `json:"status"`
	CurrentStep int                 `json:"currentStep"`
	StartedAt   time.Time           `json:"startedAt"`
	FinishedAt  *time.Time          `json:"finishedAt,omitempty"`
	Error       string              `json:"error,omitempty"`
	Workspace   string              `json:"workspace,omitempty"`
	Outputs     map[string]string   `json:"outputs"`
	Logs        []LogEntry          `json:"logs"`
	StepTimings map[int]*StepTiming `json:"stepTimings"`
}

type TaskInstance struct {
	ID              string             `json:"id"`
	Meta            TaskInstanceMeta   `json:"meta"`
	Status          TaskInstanceStatus `json:"status"`
	Generation      int64              `json:"generation"`
	ResourceVersion int64              `json:"resourceVersion"`
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	StepID    string    `json:"stepId"`
	Message   string    `json:"message"`
}

type ParamDefinition struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Optional      bool   `json:"optional"`
	RegexFrontend string `json:"regexFrontend"`
	RegexBackend  string `json:"regexBackend"`
	LookupCode    string `json:"lookupCode"`
}

type StepManifest struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Params       []ParamDefinition `json:"params"`
	OutputParams []ParamDefinition `json:"outputParams"`
}

type RunWorkflowRequest struct {
	Inputs  map[string]string `json:"inputs"`
	Trigger string            `json:"trigger"`
}

func (r *RunWorkflowRequest) Bind(_ *http.Request) error {
	r.Trigger = strings.TrimSpace(r.Trigger)
	return nil
}

type ProbeRequest struct {
	ProcessorID string            `json:"processorId"`
	Params      map[string]string `json:"params"`
}

func (p *ProbeRequest) Bind(_ *http.Request) error {
	p.ProcessorID = strings.TrimSpace(p.ProcessorID)
	if p.ProcessorID == "" {
		return errors.New("processorId is required")
	}
	return nil
}

type ProbeResponse struct {
	ProcessorID string            `json:"processorId"`
	Outputs     map[string]string `json:"outputs"`
}

type WorkflowSchemaResponse struct {
	Schema map[string]interface{} `json:"schema"`
}

type TaskCleanupResponse struct {
	Deleted int `json:"deleted"`
}

type TaskLogResponse struct {
	Logs       []LogEntry `json:"logs"`
	NextOffset int        `json:"nextOffset"`
}
