# Gemini 项目上下文

## 核心架构决策：分层开发范式 (Layered Architecture)

为了保证代码的可维护性与高性能，本项目严格遵循以下四层分层架构。新功能的开发必须按此顺序进行：

### 1. 模型层 (Models) - `pkg/models/`
- **职责**：定义所有业务实体、API 请求/响应 DTO。
- **规范**：
  - 所有结构体字段必须带有 camelCase 格式的 `json` 标签。
  - **请求模型**：必须实现 `render.Binder` 接口（提供 `Bind(*http.Request) error` 方法）以适配统一的请求解析流程。
- **示例**：`pkg/models/dns.go` 定义 `Domain` 和 `Record`。

### 2. 存储仓库层 (Repositories) - `pkg/repositories/{module}/`
- **职责**：封装底层的 `gopkg.d7z.net/middleware/kv` 存取逻辑及缓存策略。
- **子命名空间规范**：利用 `common.DB.Child(namespace...)` 进行资源隔离（如 `system`, `audit`）。

### 3. 业务服务层 (Services) - `pkg/services/{module}/`
- **职责**：编排业务流程、执行参数校验、调用 Repository。
- **审计规范**：所有修改类操作（C/U/D）必须通过 `commonaudit.FromContext(ctx).Log(action, targetID, success)` 手动上报。
- **鉴权集成**：通过 `commonauth.PermissionsFromContext(ctx).IsAllowed(id)` 执行精确拦截。

### 4. 控制器层 (Controllers) - `pkg/controllers/`
- **职责**：解析请求、调用 Service、返回标准响应。
- **序列化规范**：统一使用 `github.com/go-chi/render`。
  - 使用 `render.Bind(r, &model)` 解析请求体。
  - 使用 `common.Success` / `common.Error` 进行响应渲染。
- **中间件规范**：
  - `AuthMiddleware`：负责身份识别（Root Session 或 ServiceAccount Token）。
  - `RequirePermission(verb, resource)`：负责资源准入，并在鉴权成功后于响应头注入 `X-Matched-Policy` 以实现权限溯源。

---

## RBAC 与权限设计规范

项目采用基于资源路径的精细化权限控制体系。

### 1. 资源层级 (Resource Hierarchy)
- **DNS 模块**：遵循 `dns/<domain>/<host>/<type>` 格式。
  - 示例：`dns/example.com/www/A`。
- **通配符支持**：支持 `*`（当前层级）和 `**`（递归后续所有层级）。

### 2. 权限评估逻辑
- **全权匹配**：命中 `*` 或 `**` 时，`AllowedAll` 为 true。
- **实例匹配**：命中具体路径时，实例名将被加入 `AllowedInstances`。
- **溯源**：每次权限评估都会返回 `MatchedRule`，明确告知授权依据。

---

## RBAC 权限认证实现指南

项目采用“路由准入 + 业务拦截”的双层权限防御体系。

### 1. 路由准入：粗粒度拦截 (Controller 层)
在 `route.go` 中注册路由时，必须通过 `RequirePermission` 中间件声明该接口所需的基础权限。
- **职责**：检查当前用户是否具备操作该类资源的基本资格。
- **示例**：
  ```go
  // 仅允许拥有 dns 资源 list 权限的用户进入
  r.With(middlewares.RequirePermission("list", "dns")).Get("/domains", controllers.ListDomainsHandler)
  ```

### 2. 业务拦截：细粒度检查 (Service 层)
Service 层负责针对具体资源实例的精确权限判定。
- **职责**：防止“越权访问”，例如用户有权管理 A 域名，但尝试修改 B 域名。
- **实现模式**：
  ```go
  func UpdateDomain(ctx context.Context, id string, ...) {
      existing := repo.Get(id)
      // 构建精确资源路径进行校验
      resource := fmt.Sprintf("dns/%s", existing.Name)
      if !commonauth.PermissionsFromContext(ctx).IsAllowed(resource) {
          return nil, errors.New("permission denied: " + resource)
      }
      // 执行后续逻辑...
  }
  ```

### 3. 自动化测试中的权限 Mock
为了在不启动完整 RBAC 数据库的情况下测试 Service 逻辑，必须使用 `commonauth` 提供的注入工具。
- **Mock 全局权限**：
  ```go
  ctx := commonauth.WithPermissions(context.Background(), &models.ResourcePermissions{AllowedAll: true})
  ```
- **Mock 精确实例权限**：
  ```go
  ctx := commonauth.WithPermissions(context.Background(), &models.ResourcePermissions{
      AllowedInstances: []string{"dns/example.com"},
  })
  ```

---

## 前端 UI/UX 开发规范

遵循 **Material Design 3 (M3)** 交互规范及现代 Angular 开发标准。

### 1. 现代控制流 (Modern Control Flow)
- **强制规范**：禁止使用过时的 `*ngIf` 和 `*ngFor` 指令。
- **标准语法**：统一使用 Angular 17+ 的 `@if`, `@else`, `@for (item of list; track item.id)` 语法，以获得最佳的类型安全和渲染性能。

### 2. 交互式状态更新
对于无法预定义的交互式状态（如侧边栏开关、手动控制工具栏），必须遵循：
- **异步化**：使用 `requestAnimationFrame(() => { ... })` 包装更新逻辑，避免 NG0100 错误。

---

## 测试框架与质量保证

项目建立了完善的自动化测试体系，确保功能逻辑的正确性。

### 1. 后端功能测试 - `backend/tests/unit/`
- **环境隔离**：所有测试必须通过 `tests.SetupTestDB()` 初始化 **内存数据库 (`memory://`)**，禁止污染本地存储。
- **覆盖要求**：Service 层核心业务逻辑、RBAC 权限模拟、联级删除逻辑必须具备 100% 的功能测试覆盖。

### 2. 运行测试
- 执行 `cd backend && go test ./tests/...` 运行所有后端功能验证。

---

## 核心开发工作流

### 1. 后端逻辑开发
1. 在 `models/` 定义数据结构并实现 `Bind` 接口。
2. 在 `repositories/` 实现存储。
3. 在 `services/` 编写业务逻辑及权限检查。
4. 在 `controllers/` 暴露 Handler，使用 `render.Bind` 处理输入。

### 2. API 同步与前端联动
1. 运行 `make backend-generate`：同步 Swagger 文档并更新前端 `generated/` 代码。
2. 前端开发：利用生成的强类型 Service 客户端，采用 `@if`/`@for` 编写页面。
