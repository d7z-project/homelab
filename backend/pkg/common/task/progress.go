package task

import (
	"io"
	"time"

	"homelab/pkg/models/shared"
)

// ProgressReader wraps an io.Reader to provide real-time tracking of download progress.
type ProgressReader[T shared.TaskInfo] struct {
	io.Reader
	Total   int64
	Current int64
	Task    T
	Manager *Manager[T]
	Last    time.Time
}

func (pr *ProgressReader[T]) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	if n > 0 {
		pr.Current += int64(n)
		if pr.Total > 0 {
			now := time.Now()
			if now.Sub(pr.Last) > 500*time.Millisecond {
				pr.Task.SetProgress(float64(pr.Current) / float64(pr.Total))
				pr.Manager.Save()
				pr.Last = now
			}
		}
	}
	return n, err
}

// NewProgressReader creates a new Reader that reports progress using task.Manager.
func NewProgressReader[T shared.TaskInfo](r io.Reader, total int64, task T, manager *Manager[T]) io.Reader {
	return &ProgressReader[T]{
		Reader:  r,
		Total:   total,
		Task:    task,
		Manager: manager,
		Last:    time.Now(),
	}
}
