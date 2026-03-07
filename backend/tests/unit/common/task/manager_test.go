package task_test

import (
	"context"
	common_task "homelab/pkg/common/task"
	"homelab/pkg/models"
	"homelab/tests"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type MockTask struct {
	ID        string
	Status    models.TaskStatus
	Error     string
	CreatedAt time.Time
	mu        sync.Mutex
}

func (m *MockTask) GetID() string { return m.ID }
func (m *MockTask) GetStatus() models.TaskStatus {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Status
}
func (m *MockTask) SetStatus(s models.TaskStatus) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Status = s
}
func (m *MockTask) SetError(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Error = msg
}
func (m *MockTask) GetCreatedAt() time.Time { return m.CreatedAt }

func TestTaskManager_Lifecycle_And_Run(t *testing.T) {
	// 准备真实依赖
	cleanup := tests.SetupTestDB()
	defer cleanup()

	manager := common_task.NewManager[*MockTask]("test_lock:", "test_tasks", "test", "task_db")

	taskID := "task-1"
	task := &MockTask{ID: taskID, Status: models.TaskStatusPending, CreatedAt: time.Now()}

	// 1. Add Task
	manager.AddTask(task)
	retrieved, ok := manager.GetTask(taskID)
	assert.True(t, ok)
	assert.Equal(t, models.TaskStatusPending, retrieved.GetStatus())

	// 2. Run Task (Success flow)
	manager.RunTask(context.Background(), taskID, func(ctx context.Context, task *MockTask) error {
		// 校验内部确实改成了 Running 状态
		assert.Equal(t, models.TaskStatusRunning, task.GetStatus())
		return nil
	})

	retrieved, _ = manager.GetTask(taskID)
	assert.Equal(t, models.TaskStatusSuccess, retrieved.GetStatus())
}

func TestTaskManager_Run_FailureAndPanic(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	manager := common_task.NewManager[*MockTask]("test_lock:", "test_tasks", "test", "task_db")

	taskID := "task-panic"
	task := &MockTask{ID: taskID, Status: models.TaskStatusPending}
	manager.AddTask(task)

	// Panic flow
	manager.RunTask(context.Background(), taskID, func(ctx context.Context, task *MockTask) error {
		panic("simulate unexpected death")
	})

	retrieved, _ := manager.GetTask(taskID)
	assert.Equal(t, models.TaskStatusFailed, retrieved.GetStatus())
	assert.Contains(t, retrieved.Error, "simulate unexpected death")
}

func TestTaskManager_Cancellation(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	manager := common_task.NewManager[*MockTask]("test_lock:", "test_tasks", "test", "task_db")

	taskID := "task-cancel"
	task := &MockTask{ID: taskID, Status: models.TaskStatusPending}
	manager.AddTask(task)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		manager.RunTask(context.Background(), taskID, func(ctx context.Context, task *MockTask) error {
			// 在这死等，直到 Context 收到信号
			<-ctx.Done()
			return context.Canceled
		})
	}()

	// 稍微等一等确保协程跑进去并挂起
	time.Sleep(50 * time.Millisecond)

	// 硬取消
	cancelled := manager.CancelTask(taskID)
	assert.True(t, cancelled)

	wg.Wait()

	retrieved, _ := manager.GetTask(taskID)
	assert.Equal(t, models.TaskStatusCancelled, retrieved.GetStatus())
}

func TestTaskManager_ReconcileZombieTasks(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	manager := common_task.NewManager[*MockTask]("test_lock:", "test_tasks", "test", "task_db")

	// 模拟一个假装正在跑的任务（但其实锁没被任何人拿）
	taskID := "task-zombie"
	task := &MockTask{ID: taskID, Status: models.TaskStatusRunning}
	manager.AddTask(task)

	// 没有进程真的用 RunTask 拉起它，没有锁。所以 Reconcile 应当接管并把它标为 Failed
	manager.Reconcile(context.Background())

	retrieved, _ := manager.GetTask(taskID)
	assert.Equal(t, models.TaskStatusFailed, retrieved.GetStatus())
	assert.Contains(t, retrieved.Error, "restart or node failure")
}

func TestTaskManager_CleanupHooks(t *testing.T) {
	cleanup := tests.SetupTestDB()
	defer cleanup()

	manager := common_task.NewManager[*MockTask]("test_lock:", "test_tasks", "test", "task_db")

	hookCalled := false
	manager.SetCleanupHook(func(m *MockTask) {
		hookCalled = true
	})

	// 构造一个很久以前过期的任务
	taskID := "old-task"
	task := &MockTask{ID: taskID, Status: models.TaskStatusSuccess, CreatedAt: time.Now().Add(-48 * time.Hour)}
	manager.AddTask(task)

	// 执行清理（只留 24 小时内的）
	manager.Cleanup(24 * time.Hour)

	_, ok := manager.GetTask(taskID)
	assert.False(t, ok, "old task should be cleaned up")
	assert.True(t, hookCalled, "cleanup hook should be called")
}
