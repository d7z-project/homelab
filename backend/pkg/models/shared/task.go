package shared

import "time"

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "Pending"
	TaskStatusRunning   TaskStatus = "Running"
	TaskStatusSuccess   TaskStatus = "Success"
	TaskStatusFailed    TaskStatus = "Failed"
	TaskStatusCancelled TaskStatus = "Cancelled"
)

type TaskInfo interface {
	GetID() string
	GetStatus() TaskStatus
	SetStatus(status TaskStatus)
	SetError(msg string)
	GetCreatedAt() time.Time
	GetProgress() float64
	SetProgress(progress float64)
}
