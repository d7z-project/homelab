# Gemini 项目上下文

## 核心架构决策：分层开发范式 (Layered Architecture)

为了保证代码的可维护性与高性能，本项目严格遵循以下四层分层架构。新功能的开发必须按此顺序进行：

### 1. 模型层 (Models) - `pkg/models/`
- **职责**：定义所有业务实体、API 请求/响应 DTO，并承担 **基础格式校验** 职责。
- **规范**：
  - 所有结构体字段必须带有 camelCase 格式标签。
  - **校验闭环 (Mandatory)**：必须实现 `render.Binder` 接口。
  - **Bind 职责**：负责所有非空检查、正则表达式校验（如 ID 格式 `^[a-z0-9_]+$`）、枚举范围检查及字段预处理（如 TrimSpace）。
  - **双层正则校验**：涉及动态参数 or 变量定义时，应提供 `RegexFrontend` (UI 实时反馈) 与 `RegexBackend` (执行前强制校验)。若输入包含 `${{ ... }}` 变量语法，前端校验应自动跳过以允许动态注入。

### 2. 存储仓库层 (Repositories) - `pkg/repositories/{module}/`
- **职责**：封装底层的 KV 存取逻辑及缓存策略。
- **子命名空间规范**：利用 `common.DB.Child(namespace...)` 进行资源隔离（如 `system`, `audit`）。

### 3. 业务服务层 (Services) - `pkg/services/{module}/`
- **职责**：编排业务流程、执行 **深度业务逻辑校验**、维护资源一致性。
- **校验规范**：Service 层方法在执行任何逻辑前，**必须显式调用** `model.Bind(nil)`。
- **更新一致性 (Critical)**：在执行 `Update*` 操作时，必须显式执行 `model.ID = id`（将 URL 路径中的 ID 强制覆盖至 Body 结构体），以防止因 ID 缺失或不一致导致产生冗余记录。
- **引用合法性校验 (Mandatory)**：在创建或更新包含外键引用（如 `ServiceAccountID`, `RoleIDs`, `DomainID`）的资源时，Service 层必须在持久化前 **强制校验** 被引用资源的存在性。
- **删除完整性 (Cascading & Protection)**：
  - **级联删除**：父级资源（如 `Domain`, `Workflow`）被删除时，其所属的子资源（如 `Record`, `TaskInstance`）必须通过 Repository 同步清理。
  - **引用保护**：被其他核心模块引用的基础身份资源（如正被工作流引用的 `ServiceAccount`），在解除引用关系前 **严禁删除**。
- **任务执行逻辑 (Executor)**:
  - **步骤快照 (Consistency)**: 任务启动时必须将 Workflow 的当前步骤列表完整深度克隆至 `TaskInstance.Steps`。执行引擎必须且仅能从快照中读取指令，以确保在工作流修改或删除后，历史记录的执行逻辑仍可准确回溯。
  - **身份模拟 (Impersonation)**: 任务执行阶段必须通过 `commonauth.WithAuth` 注入工作流指定的 `ServiceAccountID`。处理器内部的所有子操作（包括 Repo 调用及跨模块 API）必须严格受限于该 SA 的权限，而非任务触发者的权限。
- **文件操作规范 (VFS)**：所有涉及业务数据及临时空间的读写必须分别通过 `common.FS` 或 `common.TempDir` (均为 `afero.Fs`) 执行。严禁直接使用 `os` 包或进行物理路径拼接。
- **模块级沙箱 (Scoped FS)**：子模块若需管理子目录，必须基于全局 Fs 通过 `afero.NewBasePathFs` 建立包级私有 Fs（如 `actionsFS`）。业务逻辑应直接面向私有 Fs 的根路径操作。
- **任务执行上下文 (`TaskContext`)**：其 `Workspace` 必须为 `afero.Fs` 类型，且必须通过 `afero.NewBasePathFs` 严格限制在任务私有目录内。
- **日志操作规范**: 
  - **分片存储**: 任务日志必须按步骤索引分片存储（`{id}.{index}.log`），其中 `0` 为初始化，`N+1` 为清理。**分片日志中严禁输出冗余的步骤启停文本，仅记录处理器产生的核心信息。**
  - **增量查询**: 必须支持基于行偏移量 (`offset`) 的增量拉取，以优化实时日志性能。
  - **耗时追踪**: 必须实时维护 `StepTimings` 记录每一分片的 `StartedAt` 与 `FinishedAt`。
  - **状态同步**: 步骤切换时必须立即持久化 `CurrentStep`，确保前端 UI 能够精准同步执行进度。
  - **自动清理**: 任务实例或父级工作流删除时，必须级联清理所有分片日志。
- **智能资源清理**: 执行自愈或清理逻辑前，必须核查数据库状态。仅当关联任务处于终态（Failed/Success）或记录不存在时，才允许物理/逻辑删除对应的虚拟目录。
- **资源清理规范**：所有任务必须在 `run` 函数中使用 `defer` 确保在任何退出场景下执行逻辑清理。
- **审计规范**：所有修改类操作（C/U/D）及触发类操作（Trigger）必须手动上报审计日志。

---

## 核心基础设施 (Core Infrastructure)

### 1. 资源初始化模式
- **存储 (DB)**: `common.DB` (kv.KV)，支持 `memory://`, `etcd://`, `redis://`。
- **锁 (Locker)**: `common.Locker` (lock.Locker)，支持 `memory://`, `etcd://`。
- **虚拟文件系统 (VFS)**: 
  - **业务存储 (`common.FS`)**: 默认 `memory://`。
  - **临时空间 (`common.TempDir`)**: 默认 `memory://`。
  - **初始化安全**: `common.InitVFS` 强制使用 `BasePathFs` 封装。`local://` 模式下严禁隐式回退，必须提供显式路径。
  - **启动自检**: 系统启动时必须对上述两个 Fs 执行“随机文件写入擦除”冒烟测试。

### 2. 鉴权与安全架构 (AuthN & AuthZ)
- **SA 令牌哈希化 (Static Security)**: Service Account 令牌在数据库中 **严禁明文存储**。必须且仅在创建或重置时向用户下发一次明文 JWT，数据库中仅持久化其 SHA-256 哈希指纹。校验时执行实时哈希比对。
- **实例级隔离 (Instance-level AuthZ)**: `ResourcePermissions.IsAllowed` 必须支持基于前缀的层级化匹配（如 `actions` 权限可覆盖 `actions/wf-1`）。
- **语义化错误分发 (Error Handling)**:
  - 控制器层严禁直接通过 `500` 错误透传权限异常。
  - **统一出口**: 必须通过 `controllers.HandleError(w, r, err)` 分发响应。
  - **类型断言**: 业务层应使用 `%w` 包装 `commonauth.ErrPermissionDenied` 等自定义错误。HandleError 会自动将其映射为标准的 `403 Forbidden` 响应。
- **Context 健壮性**: `PermissionsFromContext` 必须实现零值安全，在 Context 缺失权限对象时返回默认的空权限对象而非 `nil`，以支持安全的链式调用。

### 3. 生命周期与停机
- **Context 驱动**: `main.go` 采用 `signal.NotifyContext` 建立全局根上下文。
- **优雅停机**: 所有初始化函数必须响应 `ctx.Done()`；`http.Server` 停机超时统一设为 5s。

---

## 前端交互规范 (UX Standards)

遵循 **Material Design 3 (M3)** 交互规范，追求极致的视觉一致性与操作流畅度。

### 1. 页面结构标准
- **标准化页头**：所有列表类页面必须使用 `PageHeaderComponent` (`app-page-header`)。
  - 统一显示大标题、功能说明、数据总计及标准的“刷新”按钮。
  - 搜索状态、过滤组件应通过 `chips` 插槽插入页头。
- **自动化交互**:
  - **加载优化**: 对于重资源组件（如 Monaco Editor），必须实现加载状态监测（`isEditorLoading`）并展示居中的 `mat-spinner`，以消除首次加载的闪屏感。
  - **YAML 精简**: 导出 YAML 时必须执行递归清理，自动剔除所有 Go 零值（`false`, `0`, `""`）及空对象/集合，使生成的配置极致精简。
- **底部边距**：页面根容器必须包含 `pb-20` 类，以确保在不同设备上拥有统一的底部呼吸空间。

### 2. 表格与敏感数据展示
- **状态切换 (Status Toggle)**：
  - 所有带有“启用/禁用”功能的表格，其切换开关必须位于 **第一列**。
  - 统一使用 `mat-slide-toggle` 组件，并应用 `scale-75` 样式。
- **一次性凭证**: 对于无法找回的机密（如 SA 令牌），必须使用具备高强度风险提示（`warning` 图标、红色容器）的一次性展示对话框，并强制提供一键复制功能。
- **智能反馈**: 捕获 `403` 权限拒绝错误时，前端应解析错误消息并展示针对特定实例（如具体工作流）的引导性权限申请提示。

---

## 导航与菜单
- **分组策略**：系统级配置与监控功能（审计日志、会话管理等）应统一收纳于“系统管理”菜单组内。
- **权限隔离**：菜单项的显示应根据 `uiService.userType()` 进行实时权限过滤（如“管理会话”仅对 `root` 可见）。

---

## 测试框架与质量保证

### 1. 后端功能测试 - `backend/tests/unit/`
- **覆盖要求**：核心业务逻辑必须具备 100% 的功能测试覆盖。
- **安全拦截验证**: 必须包含针对 `TestSecurityInstanceLevel` 的专项验证，确保实例级权限隔离逻辑在全链路上生效。
- **关键路径验证**：必须包含针对 Update ID 一致性、物理路径清理逻辑及 RBAC 细粒度过滤的验证用例。
- **一致性与幂等性测试**: 必须在 `consistency_test.go` 中包含针对跨模块引用校验、删除保护以及高并发下 `TryLock` 拦截效果的专项验证。

---

## 核心开发工作流 (Core Workflow)

### 1. 迭代开发循环
1. **后端实现**：Models -> Repositories -> Services -> Controllers。
2. **逻辑验证**：执行 `go test ./tests/unit/...` 确保逻辑闭环。
3. **API 同步 (强制)**：运行 `make backend-gen`。
4. **全栈构建验证 (强制)**：交付前运行 `make all` 确保全栈类型定义无冲突。
