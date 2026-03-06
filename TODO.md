# 通用自动化引擎 (Shared Task Actions Engine)

## 1. 核心设计原则 (Core Principles)
- **空间隔离与虚拟化**: 每个任务实例运行期间拥有独立的 **虚拟工作目录 (Scoped Workspace)**（锚定在 `common.TempDir/orch`）。采用 `afero.Fs` 实现 100% 逻辑路径操作，任务结束自动逻辑清理。
- **参数映射 (GitHub Actions 风格)**: 
  - 支持引用前置步骤输出：`${{ steps.<step_id>.outputs.<key> }}`。
  - 支持引用工作流运行变量：`${{ vars.<var_key> }}`。
  - **可选语法**: `${{ ... ? }}` 标记，当变量不存在时解析为空字符串而非保留原样。
- **幂等性与并发控制**: 通过分布式锁 `common.Locker.TryLock` 实现触发阶段的幂等性拦截。同一工作流在同一时间只能有一个实例运行，冲突时新请求直接失败。
- **执行条件 (Conditional execution)**: 每一个步骤 support 可选的 `if` 条件，使用 `go-expr` 表达式进行逻辑判断。
- **日志分片与流式查询**: 
  - 日志按步骤 (`index`) 拆分为独立 `.log` 文件存储（`0`: 初始化, `1..N`: 各步骤, `N+1`: 结束清理）。
  - 支持 **偏移量 (Offset) 查询**，前端通过行数增量拉取日志，极大降低实时刷新的带宽消耗。
  - **自动清理**: 删除任务记录或所属工作流时，级联清理所有分片日志文件。
- **进度实时追踪**: 任务实例 (`TaskInstance`) 实时维护 `CurrentStep` 字段，确保前端 UI 在任务运行期间能精准显示当前激活的步骤。
- **幂等性与并发控制**: 

  - **身份绑定**: 任务执行时强制绑定 **ServiceAccount**，创建/更新时强制校验 SA 存在性。
  - **删除保护**: 禁止删除正被工作流引用的 ServiceAccount。
- **健壮性保障**: 
  - **超时中止**: 支持配置工作流级超时时间（默认 2h），超时自动触发 context cancel。
  - **Panic 恢复**: 引擎捕获所有执行过程中的 panic，记录堆栈信息并安全标记任务失败。
  - **智能自愈**: 启动时自动清理僵尸任务状态。仅当任务处于终态或记录缺失时，才物理删除其 VFS 临时目录。
  - **冒烟自检**: 系统启动时对所有 VFS 实例执行原子读写自检，失败则禁止启动。

---

## 2. 引擎架构规格 (Engine Specs)

### 2.1 任务上下文 (TaskContext)
```go
type TaskContext struct {
    WorkflowID     string             // 当前所属工作流 ID
    InstanceID     string             // 任务实例 ID
    Workspace      string             // 逻辑工作目录（相对于模块 Scoped FS）
    UserID         string             // 用于实时 RBAC 校验的触发者（SA ID 或 root）
    Context        context.Context    // 联动系统生命周期的取消信号
    CancelFunc     context.CancelFunc // 允许手动终止任务
    Logger         *TaskLogger        // VFS 持久化日志记录器
}
```

---

## 3. 技术实施规格 (Technical Specifications)

- **命名规范**: 变量键名 (Var Key) 和步骤 ID (Step ID) 强制遵循 `^[a-z0-9_]+$`。
- **默认处理器**:
  - `core/logger`: 输出自定义日志信息。
  - `core/sleep`: 执行期间按需休眠等待。
  - `core/fail`: 显式触发任务失败。
  - `core/workflow_call`: **同步调用子工作流**，支持变量传递与状态回传（具备循环调用检测）。
- **全栈 VFS**: 业务存储 (`common.FS`) 与 临时空间 (`common.TempDir`) 均由 URL Scheme 驱动，支持 `memory://` 实现零残留运行。
- **作用域沙箱**: 模块内部必须使用 `afero.NewBasePathFs` 进一步收窄文件操作权限。

---

## 4. 任务清单 (Action Plan)

### 第一阶段：核心引擎与模型实现
- [x] **Task 1: [Models]** 定义增强型 `Workflow`, `TaskInstance` 及变量定义模型。
- [x] **Task 2: [Engine]** 实现带变量解析、条件执行、超时控制、并发锁及 Panic 恢复的执行引擎。
- [x] **Task 3: [Audit]** 为编排模块挂载全量审计逻辑（C/U/D/Trigger）。

### 第二阶段：触发器与权限加固
- [x] **Task 4: [Triggers]** 实现 `TriggerManager`：集成 Cron 调度与 Webhook Token 认证流。
- [x] **Task 5: [RBAC]** 完成 Service 层细粒度资源过滤与显示名称/变量插值的权限隔离。
- [x] **Task 6: [Discovery]** 注册 `actions` 资源到 RBAC 发现中心，支持 ID 级权限分配。

### 第三阶段：架构虚拟化与一致性重构 (NEW)
- [x] **Task 7: [VFS]** 集成 `afero` 并实现基于 URL 的双重沙箱初始化。
- [x] **Task 8: [Lock]** 引入分布式锁，在 `RunWorkflow` 中实现非阻塞幂等性保护。
- [x] **Task 9: [Consistency]** 建立跨模块引用校验（SA/Role）与级联删除保护机制。
- [x] **Task 10: [Logging]** 实现任务日志的 VFS 持久化存储与流式解析。

### 第四阶段：UI 与交付验证
- [x] **Task 11: [UI-Run]** 开发动态变量输入弹窗，支持运行前参数注入。
- [x] **Task 12: [Tests]** 建立 `consistency_test.go` 专项测试，通过并发压力与一致性全量验证。
- [x] **Task 13: [Logs-Refactor]** 实现分片日志存储、增量偏移刷新及仿终端 UI 交互。
- [x] **Task 14: [Records-Mgmt]** 支持记录搜索、工作流分类展示及按天数批量清理功能。
- [x] **Task 15: [Validation]** 注册 `ParamDefinition` 及 `VarDefinition` 支持正则校验（分前后端参数）。

