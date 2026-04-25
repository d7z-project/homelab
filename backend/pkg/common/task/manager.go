package task

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"homelab/pkg/common"
	"log"
	"sync"
	"time"

	"homelab/pkg/models/shared"

	"gopkg.d7z.net/middleware/kv"
)

// Manager 泛型任务管理器
// 将全局 DB 持久化、分布式锁判定、状态机（Running->Success/Failed）、防僵死清理、上下文取消封装为统一能力
type Manager[T shared.TaskInfo] struct {
	mu          sync.RWMutex
	tasks       map[string]T
	activeTasks map[string]context.CancelFunc // 保存运行中任务的取消句柄
	dbPath      []string                      // common.DB.Child(dbPath...)
	dbKey       string
	lockPrefix  string
	cleanupHook func(T)
}

// NewManager 实例化框架。自动引用 common.DB，无需外部传递。

// dbPath 为 DB 的 namespace，如 "network", "site"
func NewManager[T shared.TaskInfo](lockPrefix string, dbKey string, dbPath ...string) *Manager[T] {
	m := &Manager[T]{
		tasks:       make(map[string]T),
		activeTasks: make(map[string]context.CancelFunc),
		dbPath:      dbPath,
		dbKey:       dbKey,
		lockPrefix:  lockPrefix,
	}
	m.loadTasks()
	return m
}

func (m *Manager[T]) db() kv.KV {
	return common.DB.Child(m.dbPath...)
}

func (m *Manager[T]) SetCleanupHook(hook func(T)) {
	m.cleanupHook = hook
}

// loadTasks 内部加载任务数据
func (m *Manager[T]) loadTasks() {
	m.mu.Lock()
	defer m.mu.Unlock()
	data, err := m.db().Get(context.Background(), m.dbKey)
	if err == nil && data != "" {
		_ = json.Unmarshal([]byte(data), &m.tasks)
	}
}

// Save 手动落盘（如果业务内部做了修改需要主动触发）
func (m *Manager[T]) Save() {
	m.mu.RLock()
	defer m.mu.RUnlock()
	b, _ := json.Marshal(m.tasks)
	_ = m.db().Put(context.Background(), m.dbKey, string(b), kv.TTLKeep)
}

// Reconcile 扫描所有任务，清理僵死状态 (由外层 Cron 或系统启动时调用)
func (m *Manager[T]) Reconcile(ctx context.Context) {
	m.mu.Lock()
	defer m.mu.Unlock()

	changed := false
	for _, t := range m.tasks {
		status := t.GetStatus()
		if status == shared.TaskStatusRunning || status == shared.TaskStatusPending {
			// 探测锁状态
			lockKey := fmt.Sprintf("%s:%s", m.lockPrefix, t.GetID())
			if release := common.Locker.TryLock(ctx, lockKey); release != nil {
				// 能拿到锁说明执行者已挂
				t.SetStatus(shared.TaskStatusFailed)
				t.SetError("Interrupted by system restart or node failure")
				changed = true
				release() // 主动释放探测锁
			}
		}
	}
	if changed {
		b, _ := json.Marshal(m.tasks)
		_ = m.db().Put(context.Background(), m.dbKey, string(b), kv.TTLKeep)
	}
}

// AddTask 注册一个新任务
func (m *Manager[T]) AddTask(t T) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tasks[t.GetID()] = t
	b, _ := json.Marshal(m.tasks)
	_ = m.db().Put(context.Background(), m.dbKey, string(b), kv.TTLKeep)
}

// GetTask 获取任务
func (m *Manager[T]) GetTask(id string) (T, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	t, ok := m.tasks[id]
	return t, ok
}

// CancelTask 尝试取消本节点正在执行的任务。
// 如果该任务正由本节点调度执行，将触发其 Context 的 Done 信号并返回 true。
// 如果任务不在本节点活跃，它只会将 KV 状态标记为 Cancelled（以便异地执行节点轮询或最终阻断）。
func (m *Manager[T]) CancelTask(id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. 发送硬取消信号（打断 IO/休眠）
	if cancel, ok := m.activeTasks[id]; ok {
		cancel()
	}

	// 2. 将状态标记为 Cancelled 更新到底层 (异地节点依赖此状态探测)
	if t, ok := m.tasks[id]; ok {
		status := t.GetStatus()
		if status == shared.TaskStatusPending || status == shared.TaskStatusRunning {
			t.SetStatus(shared.TaskStatusCancelled)
			b, _ := json.Marshal(m.tasks)
			_ = m.db().Put(context.Background(), m.dbKey, string(b), kv.TTLKeep)
			return true
		}
	}
	return false
}

// DeleteTask 删除单一任务并触发资源清理
func (m *Manager[T]) DeleteTask(id string) {
	m.CancelTask(id) // 确保若还在运行则中断

	m.mu.Lock()
	defer m.mu.Unlock()
	if t, ok := m.tasks[id]; ok {
		if m.cleanupHook != nil {
			m.cleanupHook(t)
		}
		delete(m.tasks, id)
		b, _ := json.Marshal(m.tasks)
		_ = m.db().Put(context.Background(), m.dbKey, string(b), kv.TTLKeep)
	}
}

func (m *Manager[T]) DeleteTasksByPrefix(prefix string) {
	// 先找出需要取消和删除的
	m.mu.RLock()
	var toDelete []string
	for id := range m.tasks {
		if len(id) >= len(prefix) && id[:len(prefix)] == prefix {
			toDelete = append(toDelete, id)
		}
	}
	m.mu.RUnlock()

	for _, id := range toDelete {
		m.CancelTask(id)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	changed := false
	for _, id := range toDelete {
		if t, ok := m.tasks[id]; ok {
			if m.cleanupHook != nil {
				m.cleanupHook(t)
			}
			delete(m.tasks, id)
			changed = true
		}
	}
	if changed {
		b, _ := json.Marshal(m.tasks)
		_ = m.db().Put(context.Background(), m.dbKey, string(b), kv.TTLKeep)
	}
}

// Cleanup 回收超过设定生存时长的旧任务
func (m *Manager[T]) Cleanup(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	changed := false
	for id, t := range m.tasks {
		status := t.GetStatus()
		createdAt := t.GetCreatedAt()
		if now.Sub(createdAt) > maxAge && status != shared.TaskStatusRunning {
			if m.cleanupHook != nil {
				m.cleanupHook(t)
			}
			delete(m.tasks, id)
			changed = true
		}
	}
	if changed {
		b, _ := json.Marshal(m.tasks)
		_ = m.db().Put(context.Background(), m.dbKey, string(b), kv.TTLKeep)
	}
}

// StartCleanupTimer 在后台运行清理循环
func (m *Manager[T]) StartCleanupTimer(maxAge time.Duration, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		for range ticker.C {
			m.Cleanup(maxAge)
		}
	}()
}

func (m *Manager[T]) RangeAll() []T {
	m.mu.RLock()
	defer m.mu.RUnlock()

	res := make([]T, 0, len(m.tasks))
	for _, t := range m.tasks {
		res = append(res, t)
	}
	return res
}

// RunTask 框架核心：负责调度执行任务。
// 它封装了：分布式锁防并发、状态机流转 (Pending/Cancelled -> Running -> Success/Failed/Cancelled)、上下文取消 (Context Cancel) 以及 Panic 恢复。
func (m *Manager[T]) RunTask(ctx context.Context, id string, fn func(ctx context.Context, t T) error) {
	t, ok := m.GetTask(id)
	if !ok {
		return
	}

	// 1. 获取分布式锁，证明该节点正在处理该任务
	lockKey := fmt.Sprintf("%s:%s", m.lockPrefix, t.GetID())
	release := common.Locker.TryLock(ctx, lockKey)
	if release == nil {
		log.Printf("Task %s is already handled by another node", id)
		return
	}
	defer release() //退出时自动释放锁

	// 2. 状态检查（可能在排队阶段已被要求取消）
	if t.GetStatus() == shared.TaskStatusCancelled {
		m.Save()
		return
	}

	t.SetStatus(shared.TaskStatusRunning)
	m.Save()

	// 3. 构建可取消的上下文并注册到活跃表
	taskCtx, cancel := context.WithCancel(ctx)

	m.mu.Lock()
	m.activeTasks[id] = cancel
	m.mu.Unlock()

	// 4. 兜底清理：撤销活跃注册、处理 Panic
	defer func() {
		cancel() // 防止泄露
		m.mu.Lock()
		delete(m.activeTasks, id)
		m.mu.Unlock()

		if r := recover(); r != nil {
			t.SetStatus(shared.TaskStatusFailed)
			t.SetError(fmt.Sprintf("panic: %v", r))
			m.Save()
		}
	}()

	// 5. 将具备打断能力的 Context 传递给业务核心逻辑执行
	err := fn(taskCtx, t)

	// 6. 执行完成，根据 Context 的错误状态或业务自身返回决定最终命宿
	m.mu.RLock()
	latestStatus := t.GetStatus() // 有可能其他协程调用 CancelTask 已经标记成了 Cancelled
	m.mu.RUnlock()

	if latestStatus == shared.TaskStatusCancelled || errors.Is(taskCtx.Err(), context.Canceled) {
		t.SetStatus(shared.TaskStatusCancelled)
		t.SetError("Task cancelled manually")
	} else if err != nil {
		t.SetStatus(shared.TaskStatusFailed)
		t.SetError(err.Error())
	} else {
		t.SetStatus(shared.TaskStatusSuccess)
	}
	m.Save()
}
