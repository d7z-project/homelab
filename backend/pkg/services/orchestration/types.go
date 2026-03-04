package orchestration

import (
	"context"
	"homelab/pkg/models"

	"github.com/spf13/afero"
)

type TaskContext struct {
	WorkflowID string
	InstanceID string
	Workspace  afero.Fs           // 作用域沙箱文件系统
	UserID     string             // 用于实时 RBAC 校验的触发者 ID
	Context    context.Context    // 用于传递取消信号
	CancelFunc context.CancelFunc // 允许手动终止任务
	Logger     *TaskLogger        // 实时流式日志记录器
}

type StepManifest = models.StepManifest

type StepProcessor interface {
	// 节点执行函数，返回 error 则流水线立即中断
	Execute(ctx *TaskContext, inputs map[string]string) (outputs map[string]string, err error)
	Manifest() StepManifest
}
