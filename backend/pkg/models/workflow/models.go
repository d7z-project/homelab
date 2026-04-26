package workflow

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"homelab/pkg/models/shared"

	"github.com/robfig/cron/v3"
)

var ActionIdRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

type VarDefinition struct {
	Description   string `json:"description"`
	Default       string `json:"default"`
	Required      bool   `json:"required"`
	RegexFrontend string `json:"regexFrontend"`
	RegexBackend  string `json:"regexBackend"`
}

type WorkflowV1Meta struct {
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

func (m *WorkflowV1Meta) Validate(_ context.Context) error {
	for k, v := range m.Vars {
		if !ActionIdRegex.MatchString(k) {
			return fmt.Errorf("invalid variable key '%s': only lowercase letters, numbers and underscores are allowed", k)
		}
		if m.CronEnabled && v.Required && v.Default == "" {
			return fmt.Errorf("cron job cannot be enabled when workflow has required variable without default: %s", k)
		}
	}
	if m.CronEnabled {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		if _, err := parser.Parse(m.CronExpr); err != nil {
			return fmt.Errorf("invalid cron expression: %v", err)
		}
	}
	if len(m.Steps) == 0 {
		return errors.New("at least one step is required")
	}
	stepIDs := make(map[string]bool)
	for i, step := range m.Steps {
		if step.ID == "" {
			return fmt.Errorf("step %d: ID is required", i+1)
		}
		if !ActionIdRegex.MatchString(step.ID) {
			return fmt.Errorf("step %d: invalid ID '%s': only lowercase letters, numbers and underscores are allowed", i+1, step.ID)
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("step %d: duplicate ID '%s'", i+1, step.ID)
		}
		stepIDs[step.ID] = true
		if step.Type == "" {
			return fmt.Errorf("step %d (%s): type is required", i+1, step.ID)
		}
	}
	return nil
}

type WorkflowV1Status struct {
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	// HasWebhookSecret reports whether a webhook secret is provisioned for this workflow.
	HasWebhookSecret bool `json:"hasWebhookSecret"`
}

type Workflow = shared.Resource[WorkflowV1Meta, WorkflowV1Status]

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

type TaskInstanceV1Meta struct {
	WorkflowID       string            `json:"workflowId"`
	Trigger          string            `json:"trigger"`
	UserID           string            `json:"userId"`
	ServiceAccountID string            `json:"serviceAccountId"`
	Inputs           map[string]string `json:"inputs"`
	Steps            []Step            `json:"steps"`
}

func (m *TaskInstanceV1Meta) Validate(_ context.Context) error {
	return nil
}

type TaskInstanceV1Status struct {
	Status      shared.TaskStatus `json:"status"`
	CurrentStep int               `json:"currentStep"`
	StartedAt   time.Time         `json:"startedAt"`
	FinishedAt  *time.Time        `json:"finishedAt,omitempty"`
	Error       string            `json:"error,omitempty"`
	// Workspace is the executor-owned runtime workspace path for the current task instance.
	Workspace string `json:"workspace,omitempty"`
	// QueueTopic records which dispatch topic accepted this instance for execution.
	QueueTopic string `json:"queueTopic,omitempty"`
	// QueueMessageID stores the queue message identifier used for dispatch tracking.
	QueueMessageID string `json:"queueMessageId,omitempty"`
	// QueuedAt records when the execution request was published to the queue.
	QueuedAt *time.Time `json:"queuedAt,omitempty"`
	// DispatchedAt records when a worker accepted the execution request from the queue.
	DispatchedAt *time.Time          `json:"dispatchedAt,omitempty"`
	Outputs      map[string]string   `json:"outputs"`
	Logs         []LogEntry          `json:"logs"`
	StepTimings  map[int]*StepTiming `json:"stepTimings"`
}

type TaskInstance = shared.Resource[TaskInstanceV1Meta, TaskInstanceV1Status]

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

type WorkflowExecuteJob struct {
	WorkflowID string `json:"workflowId"`
	InstanceID string `json:"instanceId"`
}
