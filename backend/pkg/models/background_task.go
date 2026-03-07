package models

import "time"

// TaskInfo 抽象的泛型后台任务约束
// 各模块的任务实体需要实现这些方法才能被通用任务管理器 (TaskManager) 接管
type TaskInfo interface {
	GetID() string
	GetStatus() string
	SetStatus(status string)
	SetError(msg string)
	GetCreatedAt() time.Time
}
