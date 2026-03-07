package models

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
)

var ActionIdRegex = regexp.MustCompile(`^[a-z0-9_]+$`)

// VarDefinition 描述一个工作流的输入变量
type VarDefinition struct {
	Description   string `json:"description"`   // 变量描述
	Default       string `json:"default"`       // 默认值 (可选变量的默认值为空)
	Required      bool   `json:"required"`      // 是否必填
	RegexFrontend string `json:"regexFrontend"` // 前端校验正则 (可选)
	RegexBackend  string `json:"regexBackend"`  // 后端校验正则 (可选)
}

// Workflow 代表一个预定义的任务编排模板
type Workflow struct {
	ID               string                   `json:"id"`
	Name             string                   `json:"name"`
	Description      string                   `json:"description"`
	Enabled          bool                     `json:"enabled"`          // 是否启用 (禁用时 Cron/Webhook/手动 均不可触发)
	Timeout          int                      `json:"timeout"`          // 超时时间 (秒)，默认 7200 (2h)，0 为不超时
	ServiceAccountID string                   `json:"serviceAccountId"` // 执行该工作流时使用的身份 (必填)
	CronEnabled      bool                     `json:"cronEnabled"`      // 是否启用定时触发
	CronExpr         string                   `json:"cronExpr"`         // Crontab 表达式
	WebhookEnabled   bool                     `json:"webhookEnabled"`   // 是否启用 Webhook 触发
	WebhookToken     string                   `json:"webhookToken"`     // Webhook 触发令牌
	Vars             map[string]VarDefinition `json:"vars"`             // 工作流启动时接受的变量定义
	Steps            []Step                   `json:"steps"`
	CreatedAt        time.Time                `json:"createdAt"`
	UpdatedAt        time.Time                `json:"updatedAt"`
}

func (w *Workflow) Bind(r *http.Request) error {
	w.Name = strings.TrimSpace(w.Name)
	if w.Name == "" {
		return errors.New("workflow name is required")
	}
	if w.ServiceAccountID == "" {
		return errors.New("service account is required")
	}

	// Validate Variable Keys
	for k, v := range w.Vars {
		if !ActionIdRegex.MatchString(k) {
			return fmt.Errorf("invalid variable key '%s': only lowercase letters, numbers and underscores are allowed", k)
		}
		if w.CronEnabled && v.Required && v.Default == "" {
			return fmt.Errorf("cron job cannot be enabled when workflow has required variable without default: %s", k)
		}
	}

	if w.CronEnabled {
		parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
		if _, err := parser.Parse(w.CronExpr); err != nil {
			return fmt.Errorf("invalid cron expression: %v", err)
		}
	}

	if len(w.Steps) == 0 {
		return errors.New("at least one step is required")
	}

	stepIDs := make(map[string]bool)
	for i, step := range w.Steps {
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

// Step 代表 Workflow 中的一个步骤
type Step struct {
	ID     string            `json:"id"`     // 步骤 ID，用于 ${{ steps.ID.outputs.key }}
	Type   string            `json:"type"`   // 处理器类型 (如 core/fetch/http)
	Name   string            `json:"name"`   // 步骤显示名称
	If     string            `json:"if"`     // 条件表达式 (go-expr)，为空则总是执行
	Params map[string]string `json:"params"` // 输入参数，支持模板字符串
	Fail   bool              `json:"fail"`   // 执行出错时是否继续执行后续步骤
}

type StepTiming struct {
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
}

// TaskInstance 代表一个正在执行或已完成的任务实例
type TaskInstance struct {
	ID               string              `json:"id"`
	WorkflowID       string              `json:"workflowId"`
	Status           string              `json:"status"`           // Pending, Running, Success, Failed, Cancelled
	CurrentStep      int                 `json:"currentStep"`      // 当前执行的步骤索引 (0: Init, 1..N: Steps, N+1: Final)
	Trigger          string              `json:"trigger"`          // Manual, Cron, Webhook
	UserID           string              `json:"userId"`           // 触发者 ID
	ServiceAccountID string              `json:"serviceAccountId"` // 执行该工作流时使用的身份 (Impersonation)
	Inputs           map[string]string   `json:"inputs"`           // 实际传入的变量值
	Workspace        string              `json:"workspace"`
	StartedAt        time.Time           `json:"startedAt"`
	FinishedAt       *time.Time          `json:"finishedAt,omitempty"`
	Error            string              `json:"error,omitempty"`
	Outputs          map[string]string   `json:"outputs"`     // 任务最终输出
	Logs             []LogEntry          `json:"logs"`        // 任务日志
	Steps            []Step              `json:"steps"`       // 运行时的步骤快照 (防篡改)
	StepTimings      map[int]*StepTiming `json:"stepTimings"` // 步骤执行耗时追踪
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	StepID    string    `json:"stepId"` // 关联的步骤 ID，空字符串代表引擎级日志
	Message   string    `json:"message"`
}

func (t *TaskInstance) Bind(r *http.Request) error {
	return nil
}

// ParamDefinition 描述一个参数的规格
type ParamDefinition struct {
	Name          string `json:"name"`
	Description   string `json:"description"`
	Optional      bool   `json:"optional"`      // 是否为可选参数
	RegexFrontend string `json:"regexFrontend"` // 前端校验正则 (可选)
	RegexBackend  string `json:"regexBackend"`  // 后端校验正则 (可选)
	LookupCode    string `json:"lookupCode"`    // 服务发现代号 (可选)
}

// StepManifest 描述一个节点处理器的规格
type StepManifest struct {
	ID           string            `json:"id"`           // 处理器唯一标识 (如 core/fetch/http)
	Name         string            `json:"name"`         // 显示名称
	Description  string            `json:"description"`  // 处理器功能简述
	Params       []ParamDefinition `json:"params"`       // 输入参数列表 (包含必选和可选)
	OutputParams []ParamDefinition `json:"outputParams"` // 输出参数
}

type RunWorkflowRequest struct {
	Inputs  map[string]string `json:"inputs"`
	Trigger string            `json:"trigger"` // Optional: Manual (default), API, Script, etc.
}

func (r *RunWorkflowRequest) Bind(req *http.Request) error {
	return nil
}

// TaskCleanupResponse 任务清理响应
type TaskCleanupResponse struct {
	Deleted int `json:"deleted"`
}

// TaskLogResponse 任务日志响应
type TaskLogResponse struct {
	Logs       []LogEntry `json:"logs"`
	NextOffset int        `json:"nextOffset"`
}

// WorkflowExecutePayload represents the data sent over the cluster bus to trigger a workflow.
type WorkflowExecutePayload struct {
	WorkflowID string            `json:"workflowId"`
	InstanceID string            `json:"instanceId"`
	UserID     string            `json:"userId"`
	Trigger    string            `json:"trigger"`
	Inputs     map[string]string `json:"inputs"`
}
