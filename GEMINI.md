# Gemini 项目上下文

## 核心架构决策：分层开发范式 (Layered Architecture)

为了保证代码的可维护性与高性能，本项目严格遵循以下四层分层架构。新功能的开发必须按此顺序进行：

### 1. 模型层 (Models) - `pkg/models/`
- **职责**：定义所有业务实体、API 请求/响应 DTO。
- **规范**：所有结构体字段必须带有 camelCase 格式的 `json` 标签。
- **示例**：`pkg/models/dns.go` 定义 `Domain` 和 `Record`。

### 2. 存储仓库层 (Repositories) - `pkg/repositories/{module}/`
- **职责**：封装底层的 BoltDB (KV) 存取逻辑及缓存策略。
- **缓存规范**：高频读取项必须接入 `github.com/hashicorp/golang-lru/v2`。
- **排序规范**：利用 `db.List` 的原生排序，避免在应用层进行大规模 `sort.Slice` 操作。

### 3. 业务服务层 (Services) - `pkg/services/{module}/`
- **职责**：编排业务流程、执行参数校验、调用 Repository。
- **审计规范**：所有修改类操作（C/U/D）必须通过 `commonaudit.FromContext(ctx).Log(action, targetID, success)` 手动上报审计日志。
- **鉴权集成**：通过 `commonauth.PermissionsFromContext(ctx).IsAllowed(id)` 执行针对具体资源实例的精确权限拦截。

### 4. 控制器层 (Controllers) - `pkg/controllers/`
- **职责**：解析 HTTP 请求参数、调用 Service、返回标准响应。
- **文档规范**：每个 Handler 必须编写完整的 Swaggo 注解，引用 `models.*` 中的类型。
- **中间件规范**：认证 (`AuthMiddleware`)、准入鉴权 (`RequirePermission`) 和审计注入 (`AuditMiddleware`) 统一在 `route.go` 中配置。

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
