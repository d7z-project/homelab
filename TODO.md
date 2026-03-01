# Homelab DNS 管理功能开发任务清单 (符合 GEMINI.md 规范)

## 阶段 1：后端分层架构实现 (Backend Layered Architecture)

遵循项目的分层开发范式，确保代码的可维护性、高性能及安全性。

### 1.1 模型层 (Models) - `backend/pkg/models/dns.go`
- **定义 Domain (域名) 结构体**:
  - 包含 `ID` (UUID), `Name`, `Status`, `Comments`, `CreatedAt`, `UpdatedAt`。
  - **规范**：所有字段使用 `camelCase` 的 `json` 标签。
- **定义 Record (解析记录) 结构体**:
  - 包含 `ID`, `DomainID`, `Name`, `Type` (枚举), `Value`, `TTL`, `Priority`, `Status`, `Comments`。
  - **规范**：同上，确保与前端 DTO 对齐。

### 1.2 存储仓库层 (Repositories) - `backend/pkg/repositories/dns/repo.go`
- **职责**：封装 `gopkg.d7z.net/middleware/kv` 的存取。
- **功能清单**:
  - 实现 `Domain` 和 `Record` 的 CRUD。
  - **隔离**：使用 `common.DB.Child("dns", "domains")` 和 `common.DB.Child("dns", "records")`。
  - **缓存**：高频读取（如根据 ID 获取域名）必须接入 `github.com/hashicorp/golang-lru/v2`。
  - **分页与排序**：参考 `audit/repo.go` 实现基于 `db.List` 的分页逻辑。

### 1.3 业务服务层 (Services) - `backend/pkg/services/dns/service.go`
- **职责**：编排业务流程、执行逻辑校验。
- **核心逻辑**:
  - **验证**：执行域名格式正则校验、IP 格式校验及 **CNAME 互斥校验 (RFC 1034)**。
  - **审计上报**：所有 C/U/D 操作必须通过 `commonaudit.FromContext(ctx).Log(action, targetID, success)` 手动记录。
  - **权限拦截**：在操作具体实例前，调用 `commonauth.PermissionsFromContext(ctx).IsAllowed(domainID)`。

### 1.4 控制器层 (Controllers) - `backend/pkg/controllers/dns_controller.go`
- **职责**：解析 HTTP 参数、调用 Service、返回标准响应。
- **API 定义**:
  - `Domain`: `POST /api/v1/dns/domains`, `GET /api/v1/dns/domains`, `PUT /api/v1/dns/domains/{id}`, `DELETE /api/v1/dns/domains/{id}`。
  - `Record`: `POST /api/v1/dns/records`, `GET /api/v1/dns/records`, `PUT /api/v1/dns/records/{id}`, `DELETE /api/v1/dns/records/{id}`。
- **文档规范**：编写完整的 Swaggo 注解，引用 `models.Domain` 和 `models.Record`。

### 1.5 路由注册 (`backend/route.go`)
- 注册 DNS 路由。
- **中间件挂载**：统一配置 `AuthMiddleware`, `RequirePermission("dns")` 和 `AuditMiddleware`。

---

## 阶段 2：API 同步与前端联动 (API Sync)

### 2.1 文档与客户端生成
- 执行 `make backend-generate`：更新 Swagger 文档并触发前端 API 生成。
- 执行 `make backend-build`：验证后端编译。

---

## 阶段 3：前端 UI 实现 (Frontend UI)

基于 Material Design 3 高质量交互规范。

### 3.1 导航与路由
- 在 `MainComponent` 中集成“DNS 管理”菜单。
- 在 `app.routes.ts` 注册路由。

### 3.2 核心页面 (`DnsComponent`)
- **布局**：复用 `.bg-surface-container-lowest` 和 `rounded-3xl` 卡片设计。
- **搜索交互**：集成动态浮动搜索框、毛玻璃遮罩 (`backdrop-blur`) 和搜索状态胶囊标签。
- **数据展示**：
  - 双选项卡切换域名与记录。
  - 实现基于滚动的无限加载 (Infinite Scroll)。

### 3.3 弹窗组件
- **`CreateDomainDialogComponent`**：基础校验与状态切换。
- **`CreateRecordDialogComponent`**：根据 DNS 类型动态调整表单（如 MX 优先权），集成同名多值规则提示。
- **`ConfirmDialogComponent`**：删除操作的二次确认与红色警告。
