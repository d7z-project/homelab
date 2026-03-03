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
- **校验规范**：Service 层方法在执行任何逻辑前，**必须显式调用** `model.Bind(nil)`。
- **更新一致性 (Critical)**：在执行 `Update*` 操作时，必须显式执行 `model.ID = id`（将 URL 路径中的 ID 强制覆盖至 Body 结构体），以防止因 ID 缺失或不一致导致产生冗余记录。
- **资源清理规范**：所有涉及临时物理目录的任务（如任务编排工作空间），必须在 `run` 函数中使用 `defer` 确保在任何退出场景下（含 Success/Failed/Cancelled）执行物理删除。
- **审计规范**：所有修改类操作（C/U/D）及触发类操作（Trigger）必须手动上报审计日志。
  - **更新 (U)**：`message` 必须记录发生变化的字段及其 **新旧值对照**（格式：`field: old -> new`）。
  - **触发 (Trigger)**：记录触发源（Manual/Cron/Webhook）及生成的实例 ID。

### 4. 控制器层 (Controllers) - `pkg/controllers/`
- **职责**：解析请求、调用 Service、返回标准响应。
- **序列化规范**：统一使用 `github.com/go-chi/render`。

---

## RBAC 与权限设计规范

项目采用基于资源路径的精细化权限控制体系。

### 1. 资源层级 (Resource Hierarchy)
- **DNS 模块**：`dns/<domain>/<host>/<type>`。
- **Orchestration 模块**：`orchestration/<workflow_id>`。
- **RBAC 模块**：`rbac`。

---

## 前端 UI/UX 开发规范

遵循 **Material Design 3 (M3)** 交互规范，追求极致的视觉一致性与操作流畅度。

### 1. 页面结构标准
- **标准化页头**：所有列表类页面必须使用 `PageHeaderComponent` (`app-page-header`)。
  - 统一显示大标题、功能说明、数据总计及标准的“刷新”按钮。
  - 搜索状态、过滤组件应通过 `chips` 插槽插入页头。
- **底部边距**：页面根容器必须包含 `pb-20` 类，以确保在不同设备上拥有统一的底部呼吸空间。

### 2. 表格展示规范
- **状态切换 (Status Toggle)**：
  - 所有带有“启用/禁用”功能的表格，其切换开关必须位于 **第一列**。
  - 统一使用 `mat-slide-toggle` 组件，并应用 `scale-75` 样式。
- **实时性要求**：对于状态频繁变更的监控页面（如“运行记录”），必须实现 **2s 自动刷新** 逻辑，并在组件销毁或页签切换时及时停止。

### 3. 导航与菜单
- **分组策略**：系统级配置与监控功能（审计日志、会话管理等）应统一收纳于“系统管理”菜单组内。
- **权限隔离**：菜单项的显示应根据 `uiService.userType()` 进行实时权限过滤（如“管理会话”仅对 `root` 可见）。

---

## 测试框架与质量保证

### 1. 后端功能测试 - `backend/tests/unit/`
- **覆盖要求**：核心业务逻辑必须具备 100% 的功能测试覆盖。
- **关键路径验证**：必须包含针对 Update ID 一致性、物理路径清理逻辑及 RBAC 细粒度过滤的验证用例。

---

## 核心开发工作流 (Core Workflow)

### 1. 迭代开发循环
1. **后端实现**：Models -> Repositories -> Services -> Controllers。
2. **逻辑验证**：执行 `go test ./tests/unit/...` 确保逻辑闭环。
3. **API 同步 (强制)**：运行 `make backend-gen`。
4. **全栈构建验证 (强制)**：交付前运行 `make all` 确保全栈类型定义无冲突。
