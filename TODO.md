# 通用任务编排引擎 (Shared Task Orchestration Engine)

## 1. 核心设计原则 (Core Principles)
- **空间隔离**: 每个任务实例运行期间拥有独立的 **临时工作目录 (Task Workspace)**（如 `/tmp/task_<id>`），任务结束（无论成功失败）自动物理清理。
- **参数映射 (GitHub Actions 风格)**: 节点输入支持引用前置步骤的输出，语法为 `${{ steps.<step_id>.outputs.<key> }}`。引擎负责在执行前解析模板字符串。
- **实时权限校验**: 任务执行时，节点处理器（如 Load 节点）需实时根据 `UserID` 校验目标资源的 RBAC 权限，确保自动化操作不越权。
- **主动取消 (Context Cancel)**: 引擎维护任务的 `context.CancelFunc`。支持通过 API 发送取消信号，利用 Go `context.Context` 传播并安全终止任务。
- **不重试原则**: 任务失败即停止，不引入重试逻辑，确保系统状态的确定性与代码极简。
- **启动自愈**: 后端启动钩子扫描数据库中 `Running` 状态的任务，将其标记为 `Failed`，并物理删除所有 `/tmp/task_*` 遗留目录。
- **结构化日志**: 日志不再存放于本地文件，而是作为 `[]LogEntry` 结构化存储于数据库中，支持按步骤 (`StepID`) 分类检索。
- **级联删除**: 删除 Workflow 模板时，自动物理删除所有关联的 TaskInstance 运行记录，确保数据库一致性。

---

## 2. 引擎架构规格 (Engine Specs)

### 2.1 任务上下文 (TaskContext)
```go
type TaskContext struct {
    InstanceID     string
    Workspace      string             // 临时目录路径
    UserID         string             // 用于实时 RBAC 校验的触发者 ID
    Context        context.Context    // 用于传递取消信号
    CancelFunc     context.CancelFunc // 允许手动终止任务
    Logger         *TaskLogger        // 内存切片日志记录器，支持按步骤 SetStep
}
```

### 2.2 数据模型 (Models)
```go
type LogEntry struct {
    Timestamp time.Time `json:"timestamp"`
    StepID    string    `json:"stepId"` // 关联步骤，空代表系统日志
    Message   string    `json:"message"`
}

type TaskInstance struct {
    // ... 基础字段
    Logs []LogEntry `json:"logs"` // 嵌入式存储
}
```

### 2.3 节点注册接口 (StepProcessor)
采用 **惰性加载 (Lazy Loading)**，业务模块在 `init()` 阶段向注册表提交处理器构造函数。
```go
type StepManifest struct {
    ID             string   // 步骤唯一标识 (用于 ${{ steps.ID... }})
    Name           string   // 显示名称
    RequiredParams []string // 必选参数名
    OptionalParams []string // 可选参数名
    OutputParams   []string // 该节点输出的 Key 列表
}

type StepProcessor interface {
    // 节点执行函数，返回 error 则流水线立即中断
    Execute(ctx *TaskContext, inputs map[string]string) (outputs map[string]string, err error)
    Manifest() StepManifest
}
```

---

## 3. 技术实施规格 (Technical Specifications)

- **目录管理**: 使用 `os.MkdirTemp` 创建空间，通过 `defer os.RemoveAll` 确保清理。启动时通过 `filepath.Glob("/tmp/task_*")` 进行物理清理。
- **数据流转**: 
  - 节点间统一传递 `map[string]string`。
  - 大文件处理：`Fetcher` 输出 `{"file_path": "..."}` -> `Parser` 输入引用 `${{ steps.fetch.outputs.file_path }}`。
- **探测接口 (Probe API)**: 提供独立的 `/api/v1/orchestration/probe` 接口，用于前端 Tag 预选等临时下载解析需求。
- **前端交互**: 
  - **全屏编辑器**: 采用 Material 3 全屏对话框，支持步骤 ID 自动生成与变量引用补全提示。
  - **沉浸式日志 (GitHub Actions Style)**: 
    - 采用全屏双栏布局，左侧为带动态状态图标的步骤导航，右侧为 XTerm.js 终端。
    - 移动端适配：侧边栏自动转换为顶部水平滚动步骤条。
    - 字体集成：集成 `JetBrains Mono` 字体栈，通过 `FitAddon` 实现终端尺寸自适应。

---

## 4. 任务清单 (Action Plan)

严格遵循 `GEMINI.md` 核心开发工作流规范。

### 第一阶段：后端分层实现 (Backend Implementation)
- [x] **Task 1: [Models]** 定义 `Workflow`, `TaskInstance`, `StepManifest` 模型，确保实现 `render.Binder`。
- [x] **Task 2: [Repositories]** 实现基于 `kv.Child("system", "orchestration")` 的任务持久化与存储库。
- [x] **Task 3: [Services]** 实现任务引擎执行器 (Executor)：处理参数映射 (`${{...}}`)、任务空间隔离、`context.Cancel` 以及启动自愈钩子。实现级联删除逻辑。
- [x] **Task 4: [Controllers]** 挂载任务编排路由与 `/api/v1/orchestration/probe` 接口。挂载相应的 `RequirePermission` 准入中间件。

### 第二阶段：逻辑验证与 API 同步 (Verify & Sync)
- [x] **Task 5: [Tests]** 编写针对流水线执行、参数映射、以及并发终止 (Context Cancel) 的单元测试。
- [x] **Task 6: [Generate]** 执行 `make backend-generate`，同步 OpenAPI 文档与前端类型代码。

### 第三阶段：前端实现 (Frontend Adaptation)
- [x] **Task 7: [UI-Workflow]** 开发任务编排看板列表，对标 DNS/RBAC 风格。
- [x] **Task 8: [UI-Builder]** 实现全屏工作流编辑器，支持动态表单生成、自动 ID 分配与参数引用提示。
- [x] **Task 9: [UI-Control]** 增加基于 XTerm.js 的 GitHub 风格实时日志详情页，支持全屏展示、按步骤过滤及移动端适配。

### 第四阶段：全栈交付验证 (Full-Stack Verification)
- [x] **Task 10: [Build]** 运行 `make all`，确保前后端均可正常编译通过无类型冲突。
