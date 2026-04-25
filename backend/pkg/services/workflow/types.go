package workflow

import (
	"context"
	workflowmodel "homelab/pkg/models/workflow"

	"github.com/spf13/afero"
)

type TaskContext struct {
	WorkflowID       string
	InstanceID       string
	Workspace        afero.Fs           // 作用域沙箱文件系统
	UserID           string             // 触发任务的原始用户 ID
	ServiceAccountID string             // 执行该工作流时使用的身份 (Impersonation)
	Context          context.Context    // 用于传递取消信号
	CancelFunc       context.CancelFunc // 允许手动终止任务
	Logger           *TaskLogger        // 实时流式日志记录器
}

type StepManifest = workflowmodel.StepManifest

type StepProcessor interface {
	// 节点执行函数，返回 error 则流水线立即中断
	Execute(ctx *TaskContext, inputs map[string]string) (outputs map[string]string, err error)
	Manifest() StepManifest
}
