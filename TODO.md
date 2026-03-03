# 通用任务编排引擎 (Shared Task Orchestration Engine)

## 1. 核心设计原则 (Core Principles)
- **空间隔离**: 每个任务实例运行期间拥有独立的 **临时工作目录 (Task Workspace)**（如 `/tmp/task_<id>`），任务结束（无论成功失败）自动物理清理。
- **参数映射 (GitHub Actions 风格)**: 
  - 支持引用前置步骤输出：`${{ steps.<step_id>.outputs.<key> }}`。
  - 支持引用工作流运行变量：`${{ vars.<var_key> }}`。
  - **可选语法**: `${{ ... ? }}` 标记，当变量不存在时解析为空字符串而非保留原样。
- **并发控制 (Single Instance)**: 同一工作流在同一时间只能有一个实例运行，冲突时新请求直接失败，确保资源独占性。
- **执行条件 (Conditional execution)**: 每一个步骤 support 可选的 `if` 条件，使用 `go-expr` 表达式进行逻辑判断。
- **触发器多样化**: 
  - **手动运行**: 支持通过 UI 交互式输入运行变量。
  - **定时任务 (Cron)**: 集成 `robfig/cron`，支持标准的 Crontab 表达式。
  - **外部钩子 (Webhook)**: 提供基于独有 Token 认证的异步触发接口，支持 GET/POST 传参。
- **安全性与身份**: 任务执行时强制绑定 **ServiceAccount**，所有节点处理器（如 DNS 记录创建）均以此身份权限进行实时 RBAC 校验。
- **健壮性保障**: 
  - **超时中止**: 支持配置工作流级超时时间（默认 2h），超时自动触发 context cancel。
  - **Panic 恢复**: 引擎捕获所有执行过程中的 panic，记录堆栈信息并安全标记任务失败，防止程序崩溃。
  - **启动自愈**: 启动时自动清理僵尸任务状态及物理临时目录。
- **全生命周期审计**: 所有工作流的 C/U/D 变更及每一次触发记录均记录于系统审计日志。

---

## 2. 引擎架构规格 (Engine Specs)

### 2.1 任务上下文 (TaskContext)
```go
type TaskContext struct {
    InstanceID     string
    Workspace      string             // 临时目录路径
    UserID         string             // 用于实时 RBAC 校验的触发者（SA ID 或 root）
    Context        context.Context    // 用于传递取消信号或处理超时
    CancelFunc     context.CancelFunc // 允许手动终止任务
    Logger         *TaskLogger        // 结构化日志记录器
}
```

---

## 3. 技术实施规格 (Technical Specifications)

- **命名规范**: 变量键名 (Var Key) 和步骤 ID (Step ID) 强制遵循 `^[a-z0-9_]+$` 限制，确保模板解析路径唯一且无歧义。
- **数据校验**: 遵循 `models.Bind` 标准进行基础格式校验。
- **前端交互**: 
  - **响应式操作**: 大屏幕显示快捷图标，小屏幕收纳至菜单。
  - **运行弹窗**: 动态生成表单，支持运行前预填默认变量值。
  - **状态切换**: Table 内集成乐观更新的启用/禁用开关。

---

## 4. 任务清单 (Action Plan)

### 第一阶段：核心引擎与模型实现
- [x] **Task 1: [Models]** 定义增强型 `Workflow`, `TaskInstance` 及变量定义模型。
- [x] **Task 2: [Engine]** 实现带变量解析、条件执行、超时控制、并发锁及 Panic 恢复的执行引擎。
- [x] **Task 3: [Audit]** 为编排模块挂载全量审计逻辑（C/U/D/Trigger）。

### 第二阶段：触发器与权限加固
- [x] **Task 4: [Triggers]** 实现 `TriggerManager`：集成 Cron 调度与 Webhook Token 认证流。
- [x] **Task 5: [RBAC]** 完成 Service 层细粒度资源过滤与显示名称/变量插值的权限隔离。
- [x] **Task 6: [Discovery]** 注册 `orchestration` 资源到 RBAC 发现中心，支持 ID 级权限分配。

### 第三阶段：UI 精细化适配
- [x] **Task 7: [UI-Builder]** 升级编辑器：增加变量声明步骤、ID 命名校验、超时与 SA 配置。
- [x] **Task 8: [UI-Run]** 开发动态变量输入弹窗，支持运行前参数注入。
- [x] **Task 9: [UI-Board]** 优化管理看板：支持 Webhook 地址复制/重置、表格内状态快捷切换。

### 第四阶段：全栈交付验证
- [x] **Task 10: [Tests]** 编写覆盖 11 个核心场景的单元测试套件，通过率为 100%。
- [x] **Task 11: [Sync]** 完成 OpenAPI 同步与全栈 `make all` 构建验证。
