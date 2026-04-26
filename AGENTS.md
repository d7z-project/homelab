# Homelab 项目开发指南

## 核心架构

本项目当前采用“模块化单体 + 分层实现”的后端结构。新增或修改后端能力时，默认沿着以下链路组织代码：

`main -> runtime.App -> module -> controller -> service -> repository -> store/KV(+VFS)`

禁止 controller 直接操作 repository，禁止 module 直接承载业务逻辑，禁止跨领域 service 形成随意双向依赖。

## 后端分层约束

### 1. 入口与运行时 - `backend/main.go`, `backend/pkg/runtime/`
- `main.go` 只负责基础设施初始化、模块装配、HTTP server 生命周期。
- 基础设施依赖统一通过 `runtime.Dependencies` 和 `runtime.ModuleDeps` 下发。
- 业务代码不要绕过 `runtime` 自建全局单例；需要 DB、Lock、Queue、PubSub、FS、Registry 时，优先从 context 读取。
- 新增后端能力时，优先做成独立 `Module`，实现 `Name / Init / RegisterRoutes / Start / Stop`。

### 2. 模块层 - `backend/pkg/modules/`
- 模块负责组装 service、注册路由、在 `Start` 阶段注册 discovery / actions / cron / queue consumer / 事件订阅。
- 路由注册统一使用 `routerx.Mount` 或 `routerx.WithScope`，不要再手写旧式 `r.With(...).Method(...)`。
- 模块间共享的横切依赖通过 `runtime.ModuleDeps` 或显式构造参数传入，不要在 controller 里临时拼装。
- 可选模块应在 `bootstrap.go` 中通过开关统一启停，保持装配入口单一。
- 当前模块开关统一放在 `Options.Modules` 下，而不是顶层平铺字段。
- 现有配置项示例：
  - YAML: `modules.workflow`, `modules.intelligence`
  - ENV: `HOMELAB_WORKFLOW`, `HOMELAB_INTELLIGENCE`

### 3. 控制器层 - `backend/pkg/controllers/`
- controller 负责请求绑定、路径参数提取、调用 service、映射 HTTP 响应。
- JSON 绑定统一使用 `controllers.BindRequest` 或 `BindOptionalRequest`。
- 分页参数统一使用 `controllers.GetCursorParams` 或 `GetSearchCursorParams`，默认 `20`，上限 `100`。
- 非成功响应统一走 `controllers.HandleError`；成功分页响应统一走 `common.CursorSuccess`。
- 模块专属 service 依赖通过中间件注入 request context，例如 `WithIPControllerDeps`、`WithSiteControllerDeps`，不要在 handler 中自行构造 service。

### 4. 服务层 - `backend/pkg/services/`
- service 负责业务规则、权限细化、长任务编排、状态推进、事件发布、跨资源一致性。
- 路由层权限检查不是 service 层权限检查的替代。涉及实例级资源、系统任务或内部调用时，service 仍应做必要的权限校验。
- 权限拒绝错误必须包装为 `fmt.Errorf("%w: ...", commonauth.ErrPermissionDenied)`，保证 controller 能稳定映射成 `403`。
- 长时间任务必须优先复用 `pkg/common/task.Manager`，支持取消、状态持久化、重启后自愈。
- 跨节点异步执行默认通过 `queue` 分发；`task.Manager`、`workflow.Executor`、导出管理器负责运行时状态，不负责集群投递。
- 需要后台执行时，应显式考虑 `context` 生命周期；如果任务必须脱离请求存活，使用 `runtime.DetachContext` 或新的后台 context。

### 5. 仓储与存储层 - `backend/pkg/repositories/`, `backend/pkg/store/`
- repository 负责领域对象的查询与持久化，不承载业务编排。
- 当前标准资源型存储统一基于 `common.ResourceRepository` + `store.ResourceStore`。
- 新资源默认建模为 `shared.Resource[Meta, Status]`，而不是散装 KV 字段集合。
- repository 命名保持现有风格：
  - 分页查询：`ScanXxx(ctx, cursor, limit, search)`
  - 全量查询：`ScanAllXxx(ctx)`
  - 单项读写：`GetXxx / SaveXxx / DeleteXxx / UpdateXxxStatus`
- 若数据天然属于资源对象，优先落入 `ResourceStore`；只有会话、全局版本号、任务快照这类非资源数据才直接使用底层 KV namespace。

### 6. 模型层 - `backend/pkg/models/`, `backend/pkg/apis/`
- 领域模型优先使用 `shared.Resource[Meta, Status]` 组织配置与运行状态。
- `Meta` 负责配置输入，`Status` 负责运行态、时间戳、任务结果等衍生状态。
- 不要把 secret 放进普通资源的 `Meta` 或 `Status`。敏感 token、webhook secret、凭据应优先落到 `core.secret` 模块，由资源仅暴露布尔状态或引用信息。
- `Meta.Validate(context.Context)` 是当前主校验入口；不要强制要求所有模型实现 `render.Binder`。
- API DTO 与内部模型的转换应留在 controller 下的 `transform.go` 或同层辅助函数，不要把 API 结构直接下沉到 repository。
- 分页响应统一使用 `shared.PaginationResponse[T]`，游标统一为 `string`。

## 路由、鉴权与审计

### 1. 路由声明
- 使用 `routerx.Scope` 统一声明：
  - `Resource`: 权限资源前缀
  - `Audit`: 审计资源名
  - `UsesAuth`: 是否启用鉴权
  - `Extra`: 模块附加中间件，如 controller 依赖注入或 workflow runtime 注入
- 每条路由通过 `routerx.Get/Post/Put/Patch/Delete` 声明动作，如 `list/create/update/delete/execute/get/admin/simulate`。

### 2. 鉴权模型
- 当前系统存在两类身份：
  - `root`：基于 session 的人工登录
  - `sa`：ServiceAccount token
- `root` 会话需要校验 JWT、Session 是否存在、IP/UA 是否匹配。
- `sa` token 当前采用 JWT 明文签发，但实际持久化统一走 `core.secret`；资源对象只保留 `HasAuthSecret` 这类状态位，不暴露明文或哈希。
- 后台系统任务如果需要全权限，不要盲目使用 `commonauth.SystemContext()` 丢掉运行时依赖；优先在现有 context 上注入 `AuthContext` 与 `Permissions`。

### 3. 审计
- 进入受审计路由时，统一通过 `AuditMiddleware` 注入 `AuditLogger`。
- service 在关键变更、登录、撤销、清理等行为上应显式记录审计日志。
- 新功能如果会改变资源状态或执行高影响动作，应补充审计记录，而不是只依赖 HTTP access log。

## 资源注册与 Discovery

- 资源发现、lookup、动作元数据统一通过 `runtime/registry` 注册。
- 需要前端下拉、联想、动作探测、资源建议的能力时，优先注册：
  - `RegisterResource`
  - `RegisterLookup`
  - `RegisterAction`
- 模块通常在 `Start` 阶段完成注册。
- lookup 返回结果必须按当前权限过滤；“能看到”默认意味着“当前身份可用”。
- 新增可执行动作时，如果前端或 workflow 需要消费该动作，必须同步注册 action descriptor。

## 异步任务、定时任务与集群事件

### 1. 通用任务
- 同步、导出、工作流实例等长任务优先复用 `pkg/common/task.Manager`。
- 任务必须支持：
  - 状态持久化
  - 手动取消
  - 分布式锁防重
  - 重启后 `Reconcile`
- 如果任务需要跨节点分发，先通过 `queue.Publish/Consume` 交给某个节点，再进入现有 runtime；不要再用 cluster event 广播去模拟任务队列。

### 2. 定时任务
- 定时调度统一基于 `robfig/cron`。
- 集群内只允许一个节点实际执行的定时任务，必须配合分布式锁或现有的分布式 cron 封装。

### 3. 事件总线
- 集群事件通过 `common.NotifyCluster` / `RegisterEventHandler` / `StartEventLoop` 协作。
- payload 应使用结构化 JSON；不要新增依赖裸字符串 payload 的事件协议。
- 如果事件发布不应受请求取消影响，应显式切换到合适的后台 context，不要默认复用可能已取消的 `taskCtx`。

## 文件系统与工作目录

- 用户数据目录和任务临时目录统一通过 `common.InitVFS` 初始化，底层使用 `afero.Fs`。
- 持久文件写入使用 `deps.FS`，临时工作区使用 `deps.TempFS`。
- workflow、导出等会创建工作目录的能力，必须负责清理临时目录。
- 不要直接假设本地磁盘路径存在；优先通过 `afero` 抽象访问文件。

## Workflow 相关约束

- `workflow` 模块是独立运行时，不要把普通业务逻辑直接塞进 controller。
- Step processor 通过注册表统一注册；新增处理器需要补充 manifest、输入校验和输出定义。
- 工作流执行涉及：
  - 分布式锁
  - 任务实例持久化
  - workspace/log 目录
  - ServiceAccount impersonation
- 新增 workflow 功能时，要同时考虑运行时上下文、日志、取消、中断恢复和权限边界。

## 网络与安全

- 所有用户提供的下载 URL 都必须经过 `common.ValidateURL(url, allowPrivate)` 做 SSRF 校验。
- 涉及内部网络访问、HTTP processor、同步源拉取时，默认先考虑私网访问风险。
- 不要在日志、接口响应或资源持久化中暴露明文敏感 token。
- `core.secret` 的配置统一走 `Options.Secret` / `HOMELAB_SECRET`：
  - `plain`：测试或本地开发可用，不做静态加密
  - `aes256:<key>`：生产默认形式，使用 `AES-256-GCM`；`<key>` 解码后必须为 32 字节
- 需要持久化 secret 时，统一通过 `core.secret` service 写入，不要直接拼装密文或自行操作 secret 索引 KV。

## 测试与验证

- 当前测试主要位于 `backend/.../*_test.go`，使用 Go 原生测试框架，不存在 `go test ./tests/unit/...` 这一统一入口。
- 修改后端时，至少运行受影响包的 `go test ./...` 子集；如果改动了基础设施或公共抽象，优先跑整个 `backend` 测试集。
- 模块装配、资源存储、discovery、权限判断、任务恢复是高价值测试点。
- 需要模拟内存环境时，优先复用 `backend/pkg/testkit`，不要在每个测试里重复手工拼 `memory://` KV、Queue、Subscriber、`afero` 文件系统。
- repository 或 service 级测试优先使用 `testkit.NewModuleDeps(t)`，直接拿到统一的内存 `ModuleDeps`。
- 模块级测试优先使用 `testkit.StartApp(t, modules...)`，走真实的 `runtime.App -> Init -> Start` 生命周期，而不是只手工调用局部函数。
- 测试专用初始化逻辑优先用 `testkit.SeedModule(...)` 或底层 `runtime.FuncModule`，把 seed 数据、假注册或观测逻辑挂进模块生命周期，不要把初始化散落在测试主体里。
- 需要验证 discovery、queue consumer、模块启动副作用时，应优先走真实模块启动路径；只有纯计算或纯仓储逻辑才退回更轻量的单元测试。

## 开发工作流

1. 后端 Swagger 变更后运行 `make backend-gen`。
2. 前端 API 客户端同步运行 `make frontend-gen`。
3. Go 客户端同步运行 `make client-go-gen`。
4. 全栈构建验证优先运行 `make frontend-build` 与 `make backend-build`。
5. 后端测试优先在 `backend/` 下运行 `go test ./...` 或受影响包集合。

## 文档维护原则

- `AGENTS.md` 记录的是“当前仓库真实采用的约束”，不是理想化设计稿。
- 当实现已经演进，优先更新本文档去匹配稳定现状；不要继续保留明显过时的规则。
- 新增基础抽象、统一中间件、公共运行时或新的代码组织模式后，应同步补充到本文档。
