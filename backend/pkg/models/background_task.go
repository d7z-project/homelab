package models

import "time"

// TaskStatus 任务状态枚举
// @Description 任务执行状态: Pending(阻塞), Running(执行), Success(完成), Failed(失败), Cancelled(取消)
// @Enum Pending, Running, Success, Failed, Cancelled
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "Pending"
	TaskStatusRunning   TaskStatus = "Running"
	TaskStatusSuccess   TaskStatus = "Success"
	TaskStatusFailed    TaskStatus = "Failed"
	TaskStatusCancelled TaskStatus = "Cancelled"
)

// TaskInfo 抽象的泛型后台任务约束
type TaskInfo interface {
	GetID() string
	GetStatus() TaskStatus
	SetStatus(status TaskStatus)
	SetError(msg string)
	GetCreatedAt() time.Time
	GetProgress() float64
	SetProgress(progress float64)
}
