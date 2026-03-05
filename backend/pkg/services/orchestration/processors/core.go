package processors

import (
	"encoding/json"
	"fmt"
	"homelab/pkg/models"
	"homelab/pkg/services/orchestration"
	"time"
)

type LoggerProcessor struct{}

func init() {
	orchestration.Register(&LoggerProcessor{})
	orchestration.Register(&SleepProcessor{})
	orchestration.Register(&FailProcessor{})
	orchestration.Register(&WorkflowCallProcessor{})
}

func (p *LoggerProcessor) Manifest() orchestration.StepManifest {
	return orchestration.StepManifest{
		ID:          "core/logger",
		Name:        "日志输出",
		Description: "将指定的消息打印到任务日志中。",
		Params: []models.ParamDefinition{
			{Name: "message", Description: "要打印的消息内容", Optional: false},
		},
		OutputParams: []models.ParamDefinition{},
	}
}

func (p *LoggerProcessor) Execute(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
	message := inputs["message"]
	ctx.Logger.Log(message)
	return nil, nil
}

// SleepProcessor pauses execution for a given duration.
type SleepProcessor struct{}

func (p *SleepProcessor) Manifest() orchestration.StepManifest {
	return orchestration.StepManifest{
		ID:          "core/sleep",
		Name:        "休眠等待",
		Description: "暂停任务执行一段时间。",
		Params: []models.ParamDefinition{
			{
				Name:          "duration",
				Description:   "等待时长，例如 5s, 10m, 1h",
				Optional:      false,
				RegexFrontend: `^\d+[smh]$`,
				RegexBackend:  `^\d+[smh]$`,
			},
		},
		OutputParams: []models.ParamDefinition{},
	}
}

func (p *SleepProcessor) Execute(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
	d, err := time.ParseDuration(inputs["duration"])
	if err != nil {
		return nil, fmt.Errorf("invalid duration: %v", err)
	}
	ctx.Logger.Logf("Sleeping for %v...", d)
	select {
	case <-time.After(d):
		return nil, nil
	case <-ctx.Context.Done():
		return nil, ctx.Context.Err()
	}
}

// FailProcessor immediately fails the workflow.
type FailProcessor struct{}

func (p *FailProcessor) Manifest() orchestration.StepManifest {
	return orchestration.StepManifest{
		ID:          "core/fail",
		Name:        "立即失败",
		Description: "中断任务并标记为失败状态。",
		Params: []models.ParamDefinition{
			{Name: "message", Description: "失败原因描述", Optional: false},
		},
		OutputParams: []models.ParamDefinition{},
	}
}

func (p *FailProcessor) Execute(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
	return nil, fmt.Errorf("explicit failure: %s", inputs["message"])
}

// WorkflowCallProcessor calls another workflow synchronously.
type WorkflowCallProcessor struct{}

func (p *WorkflowCallProcessor) Manifest() orchestration.StepManifest {
	return orchestration.StepManifest{
		ID:          "core/workflow_call",
		Name:        "调用工作流",
		Description: "同步调用另一个工作流，并等待其执行完成。不允许自我调用。",
		Params: []models.ParamDefinition{
			{
				Name:        "workflow_id",
				Description: "要调用的目标工作流 ID",
				Optional:    false,
				LookupCode:  "orchestration/workflows",
			},
			{
				Name:          "vars",
				Description:   "传递给子工作流的变量 (JSON 对象格式)",
				Optional:      true,
				RegexFrontend: `^\{.*\}$`,
				RegexBackend:  `^\{.*\}$`,
			},
		},
		OutputParams: []models.ParamDefinition{
			{Name: "instance_id", Description: "子工作流的执行实例 ID"},
			{Name: "status", Description: "子工作流的最终执行状态"},
		},
	}
}

func (p *WorkflowCallProcessor) Execute(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
	targetID := inputs["workflow_id"]
	if targetID == ctx.WorkflowID {
		return nil, fmt.Errorf("recursion detected: a workflow cannot call itself")
	}

	// Parse optional vars
	var subVars map[string]string
	if varsJSON := inputs["vars"]; varsJSON != "" {
		if err := json.Unmarshal([]byte(varsJSON), &subVars); err != nil {
			return nil, fmt.Errorf("failed to parse vars JSON: %v", err)
		}
	}

	// Fetch target workflow
	wf, err := orchestration.GetWorkflow(ctx.Context, targetID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch target workflow %s: %v", targetID, err)
	}

	ctx.Logger.Logf("Triggering sub-workflow: %s (%s)", wf.Name, targetID)
	instanceID, err := orchestration.GlobalExecutor.Execute(ctx.Context, ctx.UserID, wf, "SubWorkflow:"+ctx.InstanceID, subVars)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger sub-workflow: %v", err)
	}

	ctx.Logger.Logf("Waiting for sub-workflow %s to complete...", instanceID)

	// Poll for completion
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Context.Done():
			return nil, ctx.Context.Err()
		case <-ticker.C:
			inst, err := orchestration.GetTaskInstance(ctx.Context, instanceID)
			if err != nil {
				return nil, fmt.Errorf("failed to poll sub-workflow status: %v", err)
			}

			if inst.Status != "Running" && inst.Status != "Pending" {
				ctx.Logger.Logf("Sub-workflow %s finished with status: %s", instanceID, inst.Status)
				outputs := map[string]string{
					"instance_id": instanceID,
					"status":      inst.Status,
				}
				if inst.Status == "Success" {
					return outputs, nil
				}
				return outputs, fmt.Errorf("sub-workflow %s failed with status: %s", instanceID, inst.Status)
			}
		}
	}
}
