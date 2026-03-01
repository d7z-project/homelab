# Gemini 项目上下文

## 核心架构决策：分层开发范式 (Layered Architecture)

为了保证代码的可维护性与高性能，本项目严格遵循以下四层分层架构。新功能的开发必须按此顺序进行：

### 1. 模型层 (Models) - `pkg/models/`
- **职责**：定义所有业务实体、API 请求/响应 DTO。
- **规范**：所有结构体字段必须带有 camelCase 格式的 `json` 标签。
- **示例**：`pkg/models/dns.go` 定义 `Domain` 和 `Record`。

### 2. 存储仓库层 (Repositories) - `pkg/repositories/{module}/`
- **职责**：封装底层的 `gopkg.d7z.net/middleware/kv` 存取逻辑及缓存策略。
- **缓存规范**：高频读取项必须接入 `github.com/hashicorp/golang-lru/v2`。
- **子命名空间规范**：利用 `common.DB.Child(namespace...)` 进行资源隔离（如 `system`, `audit`）。
- **排序规范**：利用 `db.List` 的结果进行内存分页或反向迭代，确保性能。

### 3. 业务服务层 (Services) - `pkg/services/{module}/`
- **职责**：编排业务流程、执行参数校验、调用 Repository。
- **审计规范**：所有修改类操作（C/U/D）必须通过 `commonaudit.FromContext(ctx).Log(action, targetID, success)` 手动上报审计日志。
- **鉴权集成**：通过 `commonauth.PermissionsFromContext(ctx).IsAllowed(id)` 执行针对具体资源实例的精确权限拦截。

### 4. 控制器层 (Controllers) - `pkg/controllers/`
- **职责**：解析 HTTP 请求参数、调用 Service、返回标准响应。
- **文档规范**：每个 Handler 必须编写完整的 Swaggo 注解，引用 `models.*` 中的类型。
- **中间件规范**：认证 (`AuthMiddleware`)、准入鉴权 (`RequirePermission`) 和审计注入 (`AuditMiddleware`) 统一在 `route.go` 中配置。

---

## 前端 UI/UX 开发规范

为了保证全平台体验的一致性与现代感，前端开发必须遵循以下 **Material Design 3 (M3)** 高级交互规范：

### 1. 布局与视觉风格
- **容器化设计**：列表与内容区域必须使用卡片式布局。
  - 背景色：`.bg-surface-container-lowest`。
  - 边角：大圆角设计 `rounded-3xl`。
  - 宽度控制：主内容区域通常限制为 `max-w-7xl` 并居中。
- **状态视觉化**：
  - **标签 (Chips/Labels)**：使用 M3 语义色。例如：`active` 使用绿色/Primary，`inactive` 或 `disabled` 使用灰色/Outline。
  - **徽章 (Badges)**：不同类型的资源（如 DNS 的 A/CNAME/MX）应使用区分度高的颜色徽章，提升信息扫描效率。

### 2. 高级交互组件
- **动态浮动搜索 (Floating Search)**：
  - 默认隐藏搜索框，通过右下角的 **Floating Action Button (FAB)** 唤出。
  - 搜索激活时，背景必须叠加毛玻璃遮罩 (`backdrop-blur`)。
  - **智能 FAB**：根据是否有搜索关键词自动切换图标（如 `search` ↔ `close`）和颜色（如 `secondary` ↔ `tertiary`）。
- **工具栏联动**：
  - 页面顶部工具栏应根据内容滚动动态控制吸顶状态（`toolbarSticky`）。
  - 在表格/列表顶部显示实时数据状态（总数、已加载数、当前搜索关键词胶囊）。

### 3. 数据加载与列表
- **滚动体验**：优先采用 **无限加载 (Infinite Scroll)** 配合分页 API，避免传统的点击式分页。
- **空状态**：必须提供清晰的 Empty State 提示及快速创建操作入口。

### 4. 弹窗与确认 (Dialogs)
- **表单弹窗**：使用 `mat-dialog`，支持动态校验。
- **确认弹窗**：破坏性操作（如删除）必须使用全局确认组件，并以红色警告色 (`color="warn"`) 标注风险。

---

## 防御性编程规范：解决 NG0100 错误

为了杜绝 Angular 的 `ExpressionChangedAfterItHasBeenCheckedError` (NG0100)，必须遵循以下 **“父级驱动”** 状态更新规范：

### 1. 优先使用路由驱动 (Route-Driven Data)
避免在子组件的 `ngOnInit` 中调用 `UiService` 修改全局 UI 状态。
- **推荐方案**：在 `app.routes.ts` 的路由配置中通过 `data` 属性定义 UI 配置（如工具栏阴影、吸顶）。
- **优点**：`MainComponent` 会在视图检查开始前同步监听到路由变化并应用状态，100% 避免报错且性能最高。

### 2. 交互式状态更新
对于无法预定义的交互式状态（如侧边栏开关、手动控制工具栏），必须遵循：
- **异步化**：使用 `requestAnimationFrame(() => { ... })` 包装更新逻辑。
- **同步性保障**：`UiService` 内部保持 Signal 同步，仅在触发源（如组件初始化钩子）处处理异步边界。

### 3. 示例：路由驱动配置
```typescript
// app.routes.ts
{ path: 'dns', component: DnsComponent, data: { toolbar: { shadow: false } } }

// MainComponent.ts
this.router.events.pipe(
  filter(e => e instanceof ActivationEnd),
  tap(e => this.uiService.configureToolbar(e.snapshot.data['toolbar']))
).subscribe();
```

---

## 常用工具与公共定义 - `pkg/common/`
- **`auth`**：包含 `AuthContext`、`PermissionsFromContext` 等 Context 提取工具。
- **`audit`**：包含 `AuditLogger` 手动上报工具。
- **`syncmap.go`**：提供强类型的泛型 `SyncMap` 封装。

---

## 核心开发工作流

### 1. 后端逻辑开发
1. 在 `models/` 定义数据结构。
2. 在 `repositories/` 实现存储与缓存。
3. 在 `services/` 编写业务逻辑并植入手动审计。
4. 在 `controllers/` 暴露 Handler 并添加 Swagger 注解。
5. 在 `route.go` 注册路由并挂载权限中间件。

### 2. API 同步与前端联动
1. 运行 `make backend-generate`：这会更新 Swagger 文档并自动触发前端 `npm run generate-api`。
2. 运行 `make backend-build`：验证后端编译是否通过。
3. 前端开发：在 `pages/` 下创建组件，直接引用生成的强类型 Service 客户端。

### 3. 权限控制最佳实践
- **原则**：中间件负责“能不能进”，Handler 负责“能不能动这个实例”。
- **语法**：角色资源列表支持 `dns` (全权), `dns/*` (全权) 或 `dns/example.com` (限实例)。
- **测试**：利用侧边栏的“权限模拟器”验证规则是否符合预期。
