# Gemini 项目上下文

## 核心架构决策：分层开发范式 (Layered Architecture)

为了保证代码的可维护性与高性能，本项目严格遵循以下四层分层架构。新功能的开发必须按此顺序进行：

### 1. 模型层 (Models) - `pkg/models/`

- **职责**：定义所有业务实体、API 请求/响应 DTO，并承担 **基础格式校验** 职责。
- **规范**：
  - 所有结构体字段必须带有 camelCase 格式标签。
  - **校验闭环 (Mandatory)**：必须实现 `render.Binder` 接口。
  - **Bind 职责**：负责所有非空检查、正则表达式校验（如 ID 格式 `^[a-z0-9_]+$`）、枚举范围检查及字段预处理（如 TrimSpace）。

### 2. 存储仓库层 (Repositories) - `pkg/repositories/{module}/`

- **职责**：封装底层的 KV 存取逻辑及缓存策略。
- **子命名空间规范**：利用 `common.DB.Child(namespace...)` 进行资源隔离（如 `system`, `audit`）。

### 3. 业务服务层 (Services) - `pkg/services/{module}/`

- **职责**：编排业务流程、执行 **深度业务逻辑校验**、维护资源一致性。
- **语义化错误包装 (Critical)**:
  - 权限拒绝相关的错误必须使用 `fmt.Errorf("%w: context message", commonauth.ErrPermissionDenied)` 方式抛出。
  - 严禁直接返回硬编码的 `permission denied` 字符串，以确保控制器层能精准识别并映射 HTTP 状态码。
- **服务发现规范 (Discovery)**:
  - **分页安全**: 所有注册的 `LookupFunc` 必须显式执行 `limit <= 0` 校验并提供默认值（建议 20），以彻底消除计算偏移量时的除零 Panic 风险。
  - **资源特异性动作**: 使用 `RegisterResourceWithVerbs` 时必须根据业务语义定义精准的动作集（如 Actions 必须包含 `execute`，DNS 仅保留 CRUD）。
- **任务执行逻辑 (Unified Task Framework)**:
  - **框架集成**: 后端长时间运行的任务（如导出、同步）必须继承自 `pkg/common/task` 框架，利用 `Manager` 进行状态追踪与并发限制。
  - **任务取消**: 必须在循环处理逻辑中显式监听 `taskCtx.Done()`，确保用户取消或系统重启时能及时释放资源并更新状态至 `Cancelled`。
  - **状态自愈 (Reconcile)**: 服务重启时必须调用 `manager.Reconcile(ctx)`，将由于进程崩溃导致的 `Running` 任务自动标记为 `Failed` (Node Failure)。
  - **任务进度 (Progress Tracking)**: 涉及进度评估的任务模型须实现 `models.TaskInfo` 的 `Get/SetProgress` 接口。对于长文件流下载任务，应使用 `task.NewProgressReader` 对 `io.Reader` 进行包装封装，它会自动通过 `Manager.Save()` 周期性汇报最新进度，供 API 和前端读取展示。
- **标签管理逻辑 (Tagging System)**:
  - **增量更新模式 (Incremental)**: 对于大规模数据条目的标签修改，应采用 `OldTags` (需要移除的旧标签) 与 `NewTags` (需要添加的新标签) 结合的方案，避免全量覆盖导致的并发数据丢失。
  - **系统标签防护**: 严禁用户在 UI 或 API 层面添加、修改或删除以下划线 `_` 开头的系统保留标签。

### 4. 控制器层 (Controllers) - `pkg/controllers/`

- **路由定义规范 (Idiomatic Chi)**:
  - 必须采用 `r.With(middlewares.RequirePermission(...)).Method(path, handler)` 的链式调用模式注入中间件。
  - **严禁** 在路由注册处使用手动 `http.HandlerFunc` 包装或类型断言，以保持代码整洁与类型安全。
- **统一错误出口**:
  - 所有 Handler 必须统一通过 `controllers.HandleError(w, r, err)` 分发非成功响应。
  - HandleError 负责利用 `errors.Is` 进行类型探测，实现 `401/403/500` 的自动映射。

---

## 核心基础设施 (Core Infrastructure)

### 1. 鉴权与安全架构 (AuthN & AuthZ)

- **SA 令牌哈希化**: 数据库严禁存储明文 JWT。必须仅在创建/重置时下发一次，库中仅持久化其 SHA-256 哈希指纹。
- **循环依赖防护 (Import Cycle)**:
  - 跨模块通用的请求处理工具（如 `GetIP`）必须存放于 `homelab/pkg/common` 包中。
  - **严禁** 中间件包反向导入控制器包，所有底层交互必须通过 common 包中转。
- **Context 健壮性**: `PermissionsFromContext` 必须实现“零值安全”，在缺失对象时返回默认权限对象而非 `nil`，保障链式调用安全。

### 2. 网络与 SSRF 防护

- **URL 下载规范**: 所有涉及从用户指定 URL 下载内容的逻辑，必须调用 `common.ValidateURL(url, allowPrivate)` 进行 SSRF 校验。
- **内网放行策略**: 默认禁止私有网络 IP；若业务确需（如内网镜像站），必须由管理员在配置中显式开启 `allowPrivate: true`。

### 3. 虚拟文件系统 (VFS)

- **业务存储 (`common.FS`)**: 默认 `memory://`。
- **初始化安全**: `common.InitVFS` 强制使用 `BasePathFs` 封装。

### 4. 分布式任务调度 (Cron)

- **单身执行**: 集群环境下必须使用 `common.AddDistributedCronJob` 注册任务，配合分布式锁确保同一时刻只有一个节点在执行。

---

## 前端交互规范 (UX Standards)

遵循 **Material Design 3 (M3)** 交互规范。

### 1. 自动化交互与加载

- **异步反馈**: 涉及到长时间后端异步处理的操作，必须结合 `mat-progress-bar` 实时展示任务进度，并在任务终止（成功或异常）时提供明确的 Toast/Snackbar 通知。
- **防闪屏优化**: 重资源组件（如 Monaco Editor）必须实现 `isEditorLoading` 状态监测，在渲染完成前展示居中的 `mat-spinner`。
- **数据清洗**: 导出 YAML 前必须执行递归清理，自动剔除所有 Go 零值（`false`, `0`, `""`）及空对象，确保配置极致精简。

### 2. 布局与视觉规范

- **布局间距**: 遵循 M3 的留白规范。连续的输入控件（如 Select 与 Textarea）必须保持充足的垂直间距（建议使用 `margin-bottom: 1.5rem` 或 `gap-6`）。
- **表格对齐**: 预览性质的表格内容默认采用 **水平居中 (`text-center`)** 对齐，表头加粗且使用全大写字母微缩字体以增强专业感。
- **标签排序逻辑**: 命中的标签列表必须执行 **系统标签优先排序**，即所有以 `_` 下划线开头的元数据标签、地址池/策略名称始终置顶，且视觉风格与普通标签区分（推荐使用 `SortTags` 统一处理）。

### 3. 敏感数据处理

- **一次性凭证**: 无法找回的机密必须使用具备高强度风险提示（`warning` 图标、红色容器）的一次性展示对话框，并强制提供一键复制功能。

---

## 测试框架与质量保证

### 1. 安全拦截验证

- **实例级隔离测试**: 必须在 `security_instance_test.go` 中包含针对 `Deny` 场景的专项验证，确保实例级权限隔离逻辑在全链路上生效。
- **根权限 Context**: 测试用例在调用涉及权限核查的 Service 方法时，必须通过 `tests.SetupMockRootContext()` 注入权限，严禁传入裸 Context。

### 2. 长任务健壮性

- **取消机制验证**: 凡涉及新增后台任务模式（如导出、同步）的功能，必须在单元测试中包含针对 `CancelTask` 的手动触发验证，确保 Context 链路能正确关闭。

---

## 核心开发工作流 (Core Workflow)

1. **后端实现**：Models -> Repositories -> Services -> Controllers。
2. **逻辑验证**：执行 `go test ./tests/unit/...`。
3. **API 同步**：运行 `make backend-gen`。
4. **全栈构建**：运行 `make all` 确保无冲突。
