package processors

import (
	"homelab/pkg/models"
	"homelab/pkg/services/orchestration"
)

type LoggerProcessor struct{}

func init() {
	orchestration.Register(&LoggerProcessor{})
}

func (p *LoggerProcessor) Manifest() orchestration.StepManifest {
	return orchestration.StepManifest{
		ID:          "core/logger",
		Name:        "Logger",
		Description: "将指定的消息打印到任务日志中。",
		Params: []models.ParamDefinition{
			{Name: "message", Description: "要打印的消息内容，支持变量引用", Optional: false},
		},
		OutputParams: []models.ParamDefinition{},
	}
}

func (p *LoggerProcessor) Execute(ctx *orchestration.TaskContext, inputs map[string]string) (map[string]string, error) {
	message := inputs["message"]
	ctx.Logger.Log(message)
	return nil, nil
}
