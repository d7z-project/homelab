# Gemini 项目上下文

## 核心架构决策：分层开发范式 (Layered Architecture)

为了保证代码的可维护性与高性能，本项目严格遵循以下四层分层架构。新功能的开发必须按此顺序进行：

### 1. 模型层 (Models) - `pkg/models/`
- **职责**：定义所有业务实体、API 请求/响应 DTO，并承担 **基础格式校验** 职责。
- **规范**：
  - 所有结构体字段必须带有 camelCase 格式前缀的 `json` 标签。
  - **校验闭环 (Mandatory)**：必须实现 `render.Binder` 接口（提供 `Bind(*http.Request) error` 方法）。
  - **Bind 职责**：负责所有非空检查、正则表达式校验（如 ID 格式 `^[a-z0-9_]+$`）、枚举范围检查及字段预处理（如 TrimSpace）。
- **示例**：`pkg/models/dns.go` 定义 `Domain` 和 `Record` 的 `Bind` 逻辑。

### 2. 存储仓库层 (Repositories) - `pkg/repositories/{module}/`
- **职责**：封装底层的 `gopkg.d7z.net/middleware/kv` 存取逻辑及缓存策略。
- **子命名空间规范**：利用 `common.DB.Child(namespace...)` 进行资源隔离（如 `system`, `audit`）。

### 3. 业务服务层 (Services) - `pkg/services/{module}/`
- **职责**：编排业务流程、执行 **深度业务逻辑校验**、调用 Repository。
- **校验规范**：Service 层方法在执行任何逻辑前，**必须显式调用** `model.Bind(nil)`。这确保了通过内部代码调用（如 Cron 任务）与通过 API 调用遵循完全一致的校验规则。
- **审计规范**：所有修改类操作（C/U/D）及触发类操作（Trigger）必须通过 `commonaudit.FromContext(ctx).Log(action, targetID, message, success)` 手动上报。
  - **创建 (C)**：`message` 必须包含新创建实体的核心属性。
  - **更新 (U)**：`message` 记录发生变化的字段及其 **新旧值对照**（格式：`field: old -> new`）。
  - **删除 (D)**：`message` 记录 **被删除实体的完整快照 (JSON)**。
  - **触发 (Trigger)**：记录触发源（Manual/Cron/Webhook）及生成的实例 ID。
- **权限与过滤规范**：所有 `List` 类操作必须在 Service 层根据当前上下文权限进行 **细粒度过滤**（过滤资源路径如 `orchestration/<id>`）。

### 4. 控制器层 (Controllers) - `pkg/controllers/`
- **职责**：解析请求、调用 Service、返回标准响应。
- **序列化规范**：统一使用 `github.com/go-chi/render`。
  - 使用 `render.Bind(r, &model)` 解析请求体。
  - 使用 `common.Success` / `common.Error` 进行响应渲染。
- **中间件规范**：
  - `AuthMiddleware`：负责身份识别。
  - `RequirePermission(verb, resource)`：负责资源准入（粗粒度拦截）。

---

## RBAC 与权限设计规范

项目采用基于资源路径的精细化权限控制体系。

### 1. 资源层级 (Resource Hierarchy)
- **DNS 模块**：遵循 `dns/<domain>/<host>/<type>` 格式。
- **Orchestration 模块**：遵循 `orchestration/<workflow_id>` 格式。
- **RBAC 模块**：资源标识符为 `rbac`。

---

## 前端 UI/UX 开发规范

遵循 **Material Design 3 (M3)** 交互规范及现代 Angular 开发标准。

### 1. 现代控制流 (Modern Control Flow)
- **标准语法**：统一使用 Angular 17+ 的 `@if`, `@else`, `@for (item of list; track item.id)`, `@let` 语法。

### 2. 响应式与交互
- **操作反馈**：所有修改操作（如 Switch 切换）应优先使用乐观更新，并在失败时通过 `MatSnackBar` 弹出错误并回滚状态。

---

## 测试框架与质量保证

### 1. 后端功能测试 - `backend/tests/unit/`
- **覆盖要求**：核心业务逻辑必须具备 100% 的功能测试覆盖。
- **安全测试**：包含针对非法 ID 格式（正则校验）及 RBAC 越权访问的验证用例。

---

## 核心开发工作流 (Core Workflow)

### 1. 迭代开发循环
1. **后端实现**：遵循 Models -> Repositories -> Services -> Controllers 流程。
2. **逻辑验证**：执行 `go test ./tests/unit/...` 确保业务逻辑正确。
3. **API 同步 (强制)**：运行 `make backend-gen`。
4. **全栈构建验证 (强制)**：交付前运行 `make all`。
