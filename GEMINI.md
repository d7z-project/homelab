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
- **审计规范**：所有修改类操作（C/U/D）必须通过 `commonaudit.FromContext(ctx).Log(action, targetID, message, success)` 手动上报。
  - **创建 (C)**：`message` 必须包含新创建实体的核心属性。
  - **更新 (U)**：`message` 必须记录发生变化的字段及其 **新旧值对照**（格式：`field: old -> new`）。
  - **删除 (D)**：`message` 必须记录 **被删除实体的完整快照**，以便追溯。
- **权限与过滤规范**：
  - **写入拦截**：通过 `commonauth.PermissionsFromContext(ctx).IsAllowed(resource)` 执行精确拦截。
  - **列表过滤**：所有 `List` 类操作必须在 Service 层根据当前上下文权限进行 **细粒度过滤**。仅返回用户拥有权限的资源实例，严禁在 Controller 层返回全量数据后再由前端过滤。
  - **公共功能例外**：`/dns/export` 为公共只读功能，所有有效 ServiceAccount 均可访问，但其内容受账户对各域名实例的 `dns/<domain>` 权限过滤。

### 4. 控制器层 (Controllers) - `pkg/controllers/`
- **职责**：解析请求、调用 Service、返回标准响应。
- **序列化规范**：统一使用 `github.com/go-chi/render`。
  - 使用 `render.Bind(r, &model)` 解析请求体。
  - 使用 `common.Success` / `common.Error` 进行响应渲染。
- **中间件规范**：
  - `AuthMiddleware`：负责身份识别（Root Session 或 ServiceAccount Token）。
  - `RequirePermission(verb, resource)`：负责资源准入（粗粒度），并在鉴权成功后于响应头注入 `X-Matched-Policy` 以实现权限溯源。

---

## RBAC 与权限设计规范

项目采用基于资源路径的精细化权限控制体系。

### 1. 资源层级 (Resource Hierarchy)
- **DNS 模块**：遵循 `dns/<domain>/<host>/<type>` 格式。
  - 示例：`dns/example.com/www/A`。
- **RBAC 模块**：资源标识符为 `rbac`。
- **通配符支持**：支持 `*`（当前层级）和 `**`（递归后续所有层级）。

### 2. 权限评估逻辑
- **全权匹配**：命中 `*` 或 `**` 时，`AllowedAll` 为 true。
- **实例匹配**：命中具体路径时，实例名将被加入 `AllowedInstances`。
- **溯源**：每次权限评估都会返回 `MatchedRule`，明确告知授权依据。

---

## RBAC 权限认证实现指南

项目采用“路由准入 + 业务拦截”的双层权限防御体系。

### 1. 路由准入：粗粒度拦截 (Controller 层)
在 `route.go` 中注册路由时，通过 `RequirePermission` 中间件声明该接口所需的基础权限。
- **示例**：`r.With(middlewares.RequirePermission("admin", "dns")).Group(...)`。

### 2. 业务拦截：细粒度检查 (Service 层)
Service 层负责针对具体资源实例的精确权限判定与数据过滤。
- **实现模式**：
  ```go
  func ListDomains(ctx context.Context, ...) {
      all := repo.List(...)
      perms := commonauth.PermissionsFromContext(ctx)
      // 过滤逻辑
      for _, d := range all {
          if perms.IsAllowed("dns/" + d.Name) {
              res = append(res, d)
          }
      }
      return res
  }
  ```

---

## 前端 UI/UX 开发规范

遵循 **Material Design 3 (M3)** 交互规范及现代 Angular 开发标准。

### 1. 现代控制流 (Modern Control Flow)
- **强制规范**：禁止使用过时的 `*ngIf` 和 `*ngFor` 指令。
- **标准语法**：统一使用 Angular 17+ 的 `@if`, `@else`, `@for (item of list; track item.id)` 语法。

### 2. 交互式状态更新
对于无法预定义的交互式状态（如侧边栏开关、手动控制工具栏），必须遵循：
- **异步化**：使用 `requestAnimationFrame(() => { ... })` 包装更新逻辑，避免 NG0100 错误。

---

## 测试框架与质量保证

项目建立了完善的自动化测试体系。

### 1. 后端功能测试 - `backend/tests/unit/`
- **环境隔离**：所有测试通过 `tests.SetupTestDB()` 初始化 **内存数据库 (`memory://`)**。
- **安全测试**：必须包含针对 RBAC 权限过滤、细粒度拦截及审计日志内容的验证用例。
- **覆盖要求**：核心业务逻辑必须具备 100% 的功能测试覆盖。

---

## 核心开发工作流 (Core Workflow)

### 1. 迭代开发循环
1. **后端实现**：遵循 Models -> Repositories -> Services (含权限与审计) -> Controllers 流程。
2. **逻辑验证**：执行 `go test ./tests/unit/...` 确保后端业务逻辑与权限拦截正确。
3. **API 同步 (强制)**：运行 `make backend-generate`。每次修改后端 API 或 Model 后 **必须** 执行此命令，以同步更新 Swagger 文档及前端 `generated/` 强类型客户端代码。
4. **前端适配**：利用生成的最新 API 客户端更新前端页面逻辑。
5. **全栈构建验证 (强制)**：交付前 **必须** 执行 `make all`（或分别执行 `backend-build` 与 `frontend-build`）。确保后端编译无误、API 定义一致、且前端无类型冲突导致的编译失败。

### 2. 交付标准
- [ ] 后端单元测试 100% 通过（含安全与审计测试）。
- [ ] OpenAPI 文档已同步，前端 `generated/` 代码为最新。
- [ ] **全栈构建成功**：前后端均可正常编译通过，无类型定义冲突。
- [ ] 审计日志与权限拦截已按规范实现且验证通过。

