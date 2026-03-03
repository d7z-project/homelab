package models

import (
	"net/http"
	"time"
)

// Workflow 代表一个预定义的任务编排模板
type Workflow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Steps       []Step    `json:"steps"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

func (w *Workflow) Bind(r *http.Request) error {
	return nil
}

// Step 代表 Workflow 中的一个步骤
type Step struct {
	ID     string            `json:"id"`     // 步骤 ID，用于 ${{ steps.ID.outputs.key }}
	Type   string            `json:"type"`   // 处理器类型 (如 core/fetch/http)
	Name   string            `json:"name"`   // 步骤显示名称
	Params map[string]string `json:"params"` // 输入参数，支持模板字符串
}

// TaskInstance 代表一个正在执行或已完成的任务实例
type TaskInstance struct {
	ID         string            `json:"id"`
	WorkflowID string            `json:"workflowId"`
	Status     string            `json:"status"` // Pending, Running, Success, Failed, Cancelled
	UserID     string            `json:"userId"` // 触发者 ID
	Workspace  string            `json:"workspace"`
	StartedAt  time.Time         `json:"startedAt"`
	FinishedAt *time.Time        `json:"finishedAt,omitempty"`
	Error      string            `json:"error,omitempty"`
	Outputs    map[string]string `json:"outputs"` // 任务最终输出
	Logs       []LogEntry        `json:"logs"`    // 任务日志
}

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	StepID    string    `json:"stepId"` // 关联的步骤 ID，空字符串代表引擎级日志
	Message   string    `json:"message"`
}

func (t *TaskInstance) Bind(r *http.Request) error {
	return nil
}

// StepManifest 描述一个节点处理器的规格
type StepManifest struct {
	ID             string   `json:"id"`             // 处理器唯一标识 (如 core/fetch/http)
	Name           string   `json:"name"`           // 显示名称
	RequiredParams []string `json:"requiredParams"` // 必选参数名
	OptionalParams []string `json:"optionalParams"` // 可选参数名
	OutputParams   []string `json:"outputParams"`   // 该节点输出的 Key 列表
}
