# Gemini 项目开发指南 (Gemini Development Guide)

## 核心架构：分层开发范式 (Layered Architecture)

本项目严格遵循四层分层架构。开发新功能时必须按此顺序进行，严禁跨层调用或产生循环依赖。

### 1. 模型层 (Models) - `pkg/models/`
- **职责**：定义业务实体、API DTO，承担基础格式校验。
- **分页标准**：
  - **请求**：必须使用 `PaginationRequest` 结构体。
  - **响应**：必须返回 `PaginationResponse[T]` 泛型结构。
  - **游标一致性**：`Cursor` 字段必须为 `string`。即使底层技术是字节偏移（如 VFS 预览），也必须在接口层转换为字符串，以确保生成库的类型一致性。
- **校验规范**：结构体必须实现 `render.Binder` 接口，负责非空检查、正则表达式校验（ID 格式 `^[a-z0-9_\-]+$`）、枚举范围检查及字段预处理（如 `TrimSpace`）。

### 2. 存储仓库层 (Repositories) - `pkg/repositories/{module}/`
- **职责**：封装底层的 KV 存取逻辑（基于 `common.DB`）。
- **命名规范**：
  - **分页查询**：必须命名为 `ScanXXXX(ctx, cursor, limit, search)`。
  - **全量查询**：仅限系统内部或初始化使用，命名为 `ScanAllXXXX(ctx)`。
  - **严禁** 使用 `List` 作为方法前缀，以区分游标语义与传统的切片返回。

### 3. 业务服务层 (Services) - `pkg/services/{module}/`
- **职责**：编排业务流程、执行 **深度业务逻辑校验**、维护资源一致性。
- **RBAC 精细化检查 (Critical)**：
  - **实例级隔离**：在 `Update`, `Delete`, `Preview` 操作中，必须检查特定资源 ID 的权限（如 `perms.IsAllowed("network/ip/"+id)`）。
  - **语义化错误包装**：权限拒绝相关的错误必须使用 `fmt.Errorf("%w: context message", commonauth.ErrPermissionDenied)` 方式抛出，以便控制器层精准映射 403 状态码。
- **服务发现 (Discovery)**：
  - 资源发现逻辑统一归口于 `pkg/services/discovery` 包。
  - 模块必须通过 `discovery.RegisterResourceWithVerbs` 注册资源路径及动作集（DNS 仅限 CRUD，Actions/IP/Site 需包含 `execute`）。
  - **联想过滤**：`DiscoverFunc` 必须根据 `ctx` 中的权限实时过滤返回项，实现“可见即有权”。
- **任务执行逻辑**：
  - 长时间任务必须继承自 `pkg/common/task.Manager` 框架。
  - 必须支持 `taskCtx.Done()` 监听以实现平滑取消。
  - 服务重启时需调用 `manager.Reconcile(ctx)` 实现状态自愈。

### 4. 控制器层 (Controllers) - `pkg/controllers/`
- **职责**：路由注册、参数解析、统一错误出口。
- **规范**：
  - **路由定义**：采用 `r.With(middlewares.RequirePermission(verb, resource)).Method(...)` 链式调用。
  - **分页解析**：统一使用 `getCursorParams(r)` 辅助函数（默认 20，硬封顶 100）。
  - **错误处理**：所有 Handler 必须统一通过 `controllers.HandleError(w, r, err)` 返回非成功响应。
  - **成功响应**：分页结果必须使用 `common.CursorSuccess(w, r, res)` 包装。

---

## 核心基础设施 (Core Infrastructure)

### 1. 鉴权与安全架构
- **SA 令牌哈希化**：数据库严禁存储明文 JWT。仅在创建/重置时下发一次，库中仅持久化其 SHA-256 哈希指纹。
- **系统上下文**：内部系统任务（如背景引擎）应使用 `commonauth.SystemContext()` 以获得内部操作所需的 root 权限。
- **SA 安全删除**：删除 ServiceAccount 前必须通过 `discovery.CheckSAUsage` 验证其是否正被工作流等资源引用。

### 2. 集群事件总线 (Cluster Event Bus)
- **结构化 Payload**：事件 Payload 必须在 `pkg/models/` 中定义专用结构体并带有 `json` 标签。
- **发布规范**：在异步任务回调中发布事件，**必须使用 `context.Background()`**，防止因 `taskCtx` 取消导致 Pub/Sub 静默丢包。

### 3. 网络与 SSRF 防护
- **URL 下载**：所有用户指定的下载 URL 必须调用 `common.ValidateURL(url, allowPrivate)` 进行 SSRF 校验。

---

## 前端交互规范 (Frontend UX)

遵循 **Material Design 3 (M3)** 规范。

### 1. 状态管理 (Angular Signals)
- **Signal 优先**：组件状态（loading, list, cursors）必须使用 Angular Signals (`signal`, `computed`, `effect`) 管理。
- **分页控制**：使用组件内的 `pageSize` 信号管理每页请求数量，严禁硬编码。

### 2. 列表与动态加载
- **无限滚动**：大列表数据展示必须配合容器（如 `mat-sidenav-content`）的滚动监听，实现基于游标的“滑动加载”。
- **搜索重置**：当搜索关键词或过滤标签更变时，必须重置 `nextCursor` 并清空当前数据列表。

---

## 测试框架与质量保证

### 1. 验证要点
- **实例级 RBAC**：必须编写测试模拟不同用户访问非授权实例的 `Deny` 场景。
- **游标边界**：必须验证“刚好命中最后一条记录”、“Limit=1”等极端分页 Case。
- **Discovery 联想**：必须验证带斜杠的多级路径（如 `network/dns/example.com/`）下的资源推荐准确性。

### 2. 测试工具
- 单元测试模拟集群通知必须使用 `common.TriggerEvent`。
- 测试用例必须通过 `tests.SetupMockRootContext()` 或 `SetupMockContext()` 构造。

---

## 开发工作流
1. **生成 API**：后端修改 Swagger 后运行 `make backend-gen`。
2. **同步前端**：运行 `make frontend-gen` 更新 TypeScript 客户端。
3. **构建验证**：运行 `make frontend-build` 确保全栈编译正确。
4. **回归测试**：运行 `go test ./tests/unit/...`。
