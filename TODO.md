# 通用自动化引擎 (Shared Task Actions Engine)

## 1. 核心设计原则 (Core Principles)
- **空间隔离与虚拟化**: 每个任务实例运行期间拥有独立的 **虚拟工作目录 (Scoped Workspace)**（锚定在 `common.TempDir/orch`）。采用 `afero.Fs` 实现 100% 逻辑路径操作，任务结束自动逻辑清理。
- **参数映射 (GitHub Actions 风格)**: 
  - 支持引用前置步骤输出：`${{ steps.<step_id>.outputs.<key> }}`。
  - 支持引用工作流运行变量：`${{ vars.<var_key> }}`。
  - **内置状态引用**: 支持 `${{ steps.<step_id>.status }}` 获取前置步骤执行结果（布尔值 `true`/`false`）。
  - **可选语法**: `${{ ... ? }}` 标记。当变量不存在时，执行引擎解析为空字符串；校验引擎会跳过对键名存在性的静态检查。
- **配置安全性 (Validation)**:
  - **强引用校验**: 静态阶段强制核对 `ServiceAccountID`、`LookupCode` 对应的资源是否存在。
  - **时序依赖检查**: 禁止引用尚未定义（未来）的步骤 ID。
  - **输出键名核对**: 根据处理器 Manifest 声明，严格核查 `${{ steps.ID.outputs.KEY }}` 中的 KEY 是否合法。
  - **表达式安全**: `if` 条件采用变量映射 (Variable Mapping) 注入环境变量，彻底杜绝字符串替换导致的表达式注入。
- **软失败机制 (Fail-Safe)**: 支持步骤级 `fail: true` 配置。当步骤执行失败或校验不通过时，仅记录日志并标记 `status: false`，继续执行后续步骤。
- **幂等性与并发控制**: 通过分布式锁 `common.Locker.TryLock` 实现触发阶段的幂等性拦截。同一工作流在同一时间只能有一个实例运行，冲突时新请求直接失败。
- **执行条件 (Conditional execution)**: 每一个步骤 support 可选的 `if` 条件，使用 `go-expr` 表达式进行逻辑判断。
- **日志分片与流式查询**: 
  - **结构化存储**: 采用 `logs/actions/{workflow_id}/{instance_id}/{step_index}.log` 层次化存储。
  - **分片设计**: 日志按步骤 (`index`) 拆分为独立 `.log` 文件（`0`: 初始化, `1..N`: 各步骤, `N+1`: 结束清理）。
  - **增量查询**: 支持基于偏移量 (`offset`) 的流式解析，前端实现增量拉取。
- **健壮性保障**: 
  - **超时中止**: 支持配置工作流级超时时间（默认 2h），超时自动触发 context cancel。任务状态精准识别 `Cancelled` 与 `Failed`。
  - **Panic 恢复**: 引擎捕获所有执行过程中的 panic，记录堆栈信息并安全标记任务失败。
  - **递归保护**: 拦截 `core/workflow_call` 对自身的循环调用。

---

## 2. 引擎架构规格 (Engine Specs)

### 2.1 任务上下文 (TaskContext)
```go
type TaskContext struct {
    WorkflowID     string             // 当前所属工作流 ID
    InstanceID     string             // 任务实例 ID
    Workspace      afero.Fs           // 逻辑工作目录 (Scoped FS)
    UserID         string             // 用于实时 RBAC 校验的触发者
    Context        context.Context    // 联动系统生命周期的取消信号
    CancelFunc     context.CancelFunc // 允许手动终止任务
    Logger         *TaskLogger        // VFS 持久化日志记录器
}
```

---

## 3. 技术实施规格 (Technical Specifications)

- **编辑器增强 (UX)**:
  - **双模式编辑**: 同时支持 YAML (Monaco Editor) 与图形化 (Stepper) 切换编辑。
  - **后端驱动提示**: 后端实时生成工作流 JSON Schema 供前端 Monaco Editor 实现字段、步骤模板及参数的 IntelliSense 自动补全。
  - **移动端适配**: 针对 Handset 设备优化的响应式布局与触摸友好的处理器选择器。
- **命名规范**: 变量键名 (Var Key) 和步骤 ID (Step ID) 强制遵循 `^[a-z0-9_]+$`。
- **默认处理器**:
  - `core/logger`: 输出自定义日志信息。
  - `core/sleep`: 执行期间按需休眠等待。
  - `core/fail`: 显式触发任务失败。
  - `core/workflow_call`: 同步调用子工作流，具备递归调用检测。

---

## 4. 任务清单 (Action Plan)

### 第五阶段：高级交互与配置安全 (DONE)
- [x] **Task 16: [Monaco-IntelliSense]** 实现后端驱动的 YAML 自动补全与 Schema 实时校验。
- [x] **Task 17: [Validation-Plus]** 补全输出键名存在性、时序引用及递归调用的强校验逻辑。
- [x] **Task 18: [Fail-Safe]** 实现 `fail: true` 配置支持与 `${{ steps.ID.status }}` 状态追踪。
- [x] **Task 19: [Safe-Expr]** 重构表达式引擎，采用环境变量映射彻底修复注入风险。
- [x] **Task 20: [UX-Mobile]** 完成工作流编辑器的全量移动端响应式适配。
- [x] **Task 21: [Tests-Comprehensive]** 建立综合逻辑测试，覆盖分支跳转、取消执行及多重插值场景。

