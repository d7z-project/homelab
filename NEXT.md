# 下一步开发设计与执行计划 (NEXT.md)

基于目前的重构，我们在 RBAC 中引入了更加灵活且解耦的 **细粒度权限控制机制** 和 **审计日志系统**。

以下是具体的架构调整说明和后续执行计划：

---

## 一、 细粒度资源控制架构 (Fine-Grained RBAC)

### 1. 核心目标
将权限控制分为两层：
- **全局拦截层（Middleware）**：仅进行高层级的权限准入检查，不强依赖 URL 路由结构（移除从 URL 中提取 id/name 的硬编码逻辑）。
- **精确控制层（Handler/Service）**：业务逻辑根据当前请求的具体实例，调用工具类进行精确的权限判断。

### 2. 设计细节与修改点
- **数据模型整合 (`backend/pkg/auth/rbac.go`)**
  - 使用合并语法的 `Resources` 列表定义权限，例如：
    - `*`：所有资源。
    - `dns`, `dns/*`, `dns/**`：`dns` 类别下的所有实例。
    - `dns/example.com`：仅拥有 `example.com` 实例的权限。
- **Context 权限注入 (`backend/pkg/auth/auth.go` & `rbac.go`)**
  - 新增 `ResourcePermissions` 结构体，用于存储解析后的用户权限能力（`AllowedAll` 和 `AllowedInstances`）。
  - 提供 `auth.PermissionsFromContext(ctx)` 工具类，方便下游 Handler 随时获取当前请求上下文中的权限信息。
  - 提供 `perms.IsAllowed(resourceName)` 方法，业务端只需调用此方法即可知道用户是否有权限操作特定实例。
- **鉴权中间件重构 (`backend/pkg/routers/auth.go`)**
  - **移除** 之前在 `RequirePermission` 中通过 `chi.URLParam` 获取 `id` 或 `name` 的冗余且易错的代码。
  - **机制变更**：中间件现在只负责检查用户对该 `resource` 和 `verb` 是否具有基础的准入权限（哪怕只有一个具体实例的权限）。如果通过，它会将计算出的 `ResourcePermissions` 注入到 Request Context 中，并放行请求。具体的越权拦截交由 Handler 去处理。

---

## 二、 审计日志系统 (Audit Logging) [已完成]

- **数据模型**：构建了包含 Subject, Action, Resource, TargetID 等维度的 `AuditLog`。
- **存储**：利用 BoltDB 的 `system/audit` 命名空间，采用 Timestamp-UUID 作为排序 Key，实现了高效的分页查询。
- **中间件**：`AuditMiddleware` 已成功挂载，能够拦截非 GET 请求并异步记入日志。

---

## 三、 Token 生命周期与鉴权缓存 [已完成]

- **LRU Cache**：集成了 `hashicorp/golang-lru/v2`，对 Role 和 Token 实现了高效的 LRU 缓存，辅以 `sync.RWMutex` 管理的 RoleBindings 缓存。
- **LastUsedAt 追踪**：通过 `common.SyncMap` 和内存节流（5 分钟阈值），在 `AuthMiddleware` 中实现了对 ServiceAccount 活跃时间的异步低耗更新。

---

## 四、 下一步执行动作 (Next Steps)

现在 RBAC 和 Audit 基础设施已经彻底完备，接下来将正式进入业务模块开发：

1. **DNS 核心模型开发 (`TODO.md` 阶段一)**
   - 建立 `Domain` 和 `Record` 模型。
   - 实现底层的 DB 增删改查逻辑（包含验证逻辑）。
2. **DNS API 路由集成 (`TODO.md` 阶段二)**
   - 暴露 RESTful 接口。
   - **关键**：在 DNS Handler 的 PUT/DELETE 等针对具体实例的方法中，提取出请求的域名 ID，并调用 `auth.PermissionsFromContext(r.Context()).IsAllowed(domainID)` 来做最终的安全防线。
3. **前端 UI 集成 (`TODO.md` 阶段三 & 四)**
   - 重新生成 API 客户端代码。
   - 开发前端双选项卡管理的 DNS 配置界面，复用完善的浮动搜索和确认弹窗组件。
